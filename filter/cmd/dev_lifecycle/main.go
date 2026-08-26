// Command lifecycle traces each GUFI's fdpsFlightStatus over time to answer one
// question: does a flight that goes DROPPED later come back to ACTIVE?
//
// If DROPPED -> ... -> ACTIVE is common, then DROPPED is a transition (the
// flight left this system's / center's en route coverage and resurfaced under
// the same GUFI), NOT a termination. That decides whether the database-writer
// may ever treat DROPPED as "flight over" (it must not).
//
// Like census, it reads fixm.normalized directly from partition 0 starting at
// the beginning, so it never joins a consumer group or disturbs offsets. Safe
// to run repeatedly. Each Kafka message is one envelope: a JSON array of flight
// objects, each shaped {"flight": {...}}.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// element mirrors one array entry of the normalized envelope, pulling only the
// few fields this analysis needs.
type element struct {
	Flight struct {
		Timestamp string `json:"timestamp"`
		Gufi      struct {
			Code string `json:"code"`
		} `json:"gufi"`
		FlightStatus struct {
			FDPSFlightStatus string `json:"fdpsFlightStatus"`
		} `json:"flightStatus"`
		Arrival struct {
			RunwayPositionAndTime struct {
				RunwayTime struct {
					Actual struct {
						Time string `json:"time"`
					} `json:"actual"`
				} `json:"runwayTime"`
			} `json:"runwayPositionAndTime"`
		} `json:"arrival"`
	} `json:"flight"`
}

// obs is one status observation for a GUFI, ordered by ts (idx breaks ties so
// same-timestamp messages keep read order).
type obs struct {
	ts        time.Time
	idx       int
	status    string
	hasArrOff bool // arrival runway actual time present in this message
}

func main() {
	brokers := flag.String("brokers", envOr("KAFKA_BROKERS", "localhost:9092"), "comma-separated broker list")
	topic := flag.String("topic", envOr("KAFKA_TOPIC_NORMALIZED", "fixm.normalized"), "topic to read")
	limit := flag.Int("limit", 0, "stop after N envelopes, 0 for all")
	idle := flag.Duration("idle", 5*time.Second, "stop after this long with no new messages")
	examples := flag.Int("examples", 12, "how many resurrected timelines to print")
	flag.Parse()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   strings.Split(*brokers, ","),
		Topic:     *topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10 << 20,
	})
	defer reader.Close()

	if err := reader.SetOffset(kafka.FirstOffset); err != nil {
		log.Fatalf("seek: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	timelines := map[string][]obs{}
	var envelopes, observations, idx int

	log.Printf("reading %s from %s", *topic, *brokers)

	for {
		if *limit > 0 && envelopes >= *limit {
			break
		}

		readCtx, readCancel := context.WithTimeout(ctx, *idle)
		msg, err := reader.ReadMessage(readCtx)
		readCancel()
		if err != nil {
			if ctx.Err() != nil {
				log.Println("interrupted")
			} else {
				log.Printf("done reading: %v", err)
			}
			break
		}

		envelopes++

		env, err := decodeElements(msg.Value)
		if err != nil {
			log.Printf("unmarshal failed at offset %d: %v", msg.Offset, err)
			continue
		}

		for _, e := range env {
			gufi := e.Flight.Gufi.Code
			status := e.Flight.FlightStatus.FDPSFlightStatus
			if gufi == "" || status == "" {
				continue
			}
			observations++
			timelines[gufi] = append(timelines[gufi], obs{
				ts:        parseTime(e.Flight.Timestamp),
				idx:       idx,
				status:    status,
				hasArrOff: e.Flight.Arrival.RunwayPositionAndTime.RunwayTime.Actual.Time != "",
			})
			idx++
		}

		if envelopes%2000 == 0 {
			log.Printf("%d envelopes, %d observations, %d gufis", envelopes, observations, len(timelines))
		}
	}

	report(timelines, envelopes, observations, *examples)
}

// decodeElements handles both envelope shapes: a single flight object
// {"flight": {...}} (the normal case) or a JSON array of them.
func decodeElements(data []byte) ([]element, error) {
	if trimmed := bytes.TrimLeft(data, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '[' {
		var multi []element
		if err := json.Unmarshal(data, &multi); err != nil {
			return nil, err
		}
		return multi, nil
	}
	var single element
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	return []element{single}, nil
}

func report(timelines map[string][]obs, envelopes, observations, examples int) {
	var (
		everDropped        int
		droppedThenActive  int
		droppedThenAny     = map[string]int{} // first status seen AFTER the first DROPPED
		droppedTerminalEnd int                // DROPPED with nothing after it (flight's last known state)
		droppedTotal       int
		droppedWithArrOff  int
	)

	var resurrectedExamples []string

	for gufi, seq := range timelines {
		sort.Slice(seq, func(i, j int) bool {
			if seq[i].ts.Equal(seq[j].ts) {
				return seq[i].idx < seq[j].idx
			}
			return seq[i].ts.Before(seq[j].ts)
		})

		firstDrop := -1
		sawDrop := false
		sawActiveAfterDrop := false
		for i, o := range seq {
			if o.status == "DROPPED" {
				droppedTotal++
				if o.hasArrOff {
					droppedWithArrOff++
				}
				if !sawDrop {
					sawDrop = true
					firstDrop = i
				}
			}
			if sawDrop && o.status == "ACTIVE" {
				sawActiveAfterDrop = true
			}
		}

		if sawDrop {
			everDropped++
			if firstDrop == len(seq)-1 {
				droppedTerminalEnd++
			} else {
				droppedThenAny[seq[firstDrop+1].status]++
			}
		}
		if sawActiveAfterDrop {
			droppedThenActive++
			if len(resurrectedExamples) < examples {
				resurrectedExamples = append(resurrectedExamples, formatTimeline(gufi, seq))
			}
		}
	}

	uniqueGufis := len(timelines)

	fmt.Printf("\n%s\n", strings.Repeat("=", 78))
	fmt.Printf("LIFECYCLE: does DROPPED return to ACTIVE?\n")
	fmt.Printf("%s\n", strings.Repeat("=", 78))
	fmt.Printf("envelopes read ............ %d\n", envelopes)
	fmt.Printf("flight observations ....... %d\n", observations)
	fmt.Printf("unique GUFIs .............. %d\n\n", uniqueGufis)

	fmt.Printf("GUFIs ever DROPPED ........ %d  (%.1f%% of all GUFIs)\n", everDropped, pct(everDropped, uniqueGufis))
	fmt.Printf("  ...later back to ACTIVE . %d  (%.1f%% of ever-DROPPED)  <-- the answer\n", droppedThenActive, pct(droppedThenActive, everDropped))
	fmt.Printf("  ...DROPPED was last seen  %d  (%.1f%% of ever-DROPPED)\n\n", droppedTerminalEnd, pct(droppedTerminalEnd, everDropped))

	fmt.Printf("First status AFTER the first DROPPED (across ever-DROPPED GUFIs):\n")
	for _, s := range sortedKeys(droppedThenAny) {
		fmt.Printf("  %-10s %6d  (%.1f%%)\n", s, droppedThenAny[s], pct(droppedThenAny[s], everDropped))
	}
	fmt.Printf("  %-10s %6d  (%.1f%%)\n\n", "(none)", droppedTerminalEnd, pct(droppedTerminalEnd, everDropped))

	fmt.Printf("Secondary signal: DROPPED messages carrying arrival actual (wheels-on) time:\n")
	fmt.Printf("  %d of %d DROPPED messages  (%.2f%%)\n", droppedWithArrOff, droppedTotal, pct(droppedWithArrOff, droppedTotal))
	fmt.Printf("  (near-zero => DROPPED is not a landing signal)\n\n")

	if len(resurrectedExamples) > 0 {
		fmt.Printf("Example resurrected timelines (DROPPED -> ... -> ACTIVE):\n")
		for _, e := range resurrectedExamples {
			fmt.Printf("  %s\n", e)
		}
	}
	fmt.Println()
}

func formatTimeline(gufi string, seq []obs) string {
	var parts []string
	var last string
	for _, o := range seq {
		if o.status == last {
			continue // collapse runs of the same status for readability
		}
		parts = append(parts, o.status)
		last = o.status
	}
	return fmt.Sprintf("%s : %s", gufi, strings.Join(parts, " -> "))
}

func parseTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	return keys
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
