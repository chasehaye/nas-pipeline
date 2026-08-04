// Command census reconstructs the shape of the FIXM feed from the data
// itself and prints it as an annotated tree.
//
// A flat list of paths tells you what exists but makes you rebuild the
// hierarchy mentally. This renders what the XML actually looks like:
// nesting, presence, cardinality, and a real sample value for every leaf,
// so the output can be read straight down while writing structs.
//
// Reads the partition directly rather than joining a consumer group, so it
// always starts from the beginning and never disturbs the processor's
// offsets. Safe to run repeatedly.
package main

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// Attributes worth enumerating in full. Anything not listed still gets a
// sample value and a cardinality count, just not the complete value set.
// High-cardinality fields like callsigns would be a memory leak and tell
// you nothing.
var enumerate = map[string]bool{
	"source": true, "system": true, "flightType": true,
	"fdpsFlightStatus": true, "reportSource": true, "coastIndicator": true,
	"event": true, "wakeTurbulence": true, "equipmentQualifier": true,
	"uom": true, "codeSpace": true, "invalid": true, "nil": true,
	"initialFlightRules": true, "airspaceType": true, "centre": true,
	"coordinationTimeHandling": true, "standardCapabilities": true,
	"tfmsSpecialAircraftQualifier": true, "aircraftPerformance": true,
	"phase": true, "airborneHold": true,
}

// node is one element in the reconstructed tree
type node struct {
	name string

	// flights this element appeared in at all
	flights int

	// how many times it appeared within a single flight, at most
	maxRepeat int

	// leaf text content, if any
	hasText     bool
	sampleText  string
	textValues  map[string]int

	attrs    map[string]*attr
	children map[string]*node

	// insertion order, so siblings print in the order first seen rather
	// than alphabetically -- closer to how the document reads
	order []string
}

type attr struct {
	name    string
	flights int
	sample  string
	values  map[string]int
}

func newNode(name string) *node {
	return &node{
		name:       name,
		attrs:      map[string]*attr{},
		children:   map[string]*node{},
		textValues: map[string]int{},
	}
}

type census struct {
	envelopes int
	flights   int
	root      *node
	seen   map[*node]int
	seenAt map[*attr]bool
}

func newCensus() *census {
	return &census{
		root:   newNode("flight"),
		seen:   map[*node]int{},
		seenAt: map[*attr]bool{},
	}
}

func (c *census) walk(data []byte) error {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []*node
	inFlight := false

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		switch t := tok.(type) {

		case xml.StartElement:
			if t.Name.Local == "flight" {
				c.flights++
				c.flush()
				inFlight = true
				stack = []*node{c.root}
				c.record(c.root, t.Attr)
				continue
			}
			if !inFlight {
				continue
			}

			parent := stack[len(stack)-1]
			n, ok := parent.children[t.Name.Local]
			if !ok {
				n = newNode(t.Name.Local)
				parent.children[t.Name.Local] = n
				parent.order = append(parent.order, t.Name.Local)
			}
			stack = append(stack, n)
			c.record(n, t.Attr)

		case xml.CharData:
			if !inFlight || len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			n := stack[len(stack)-1]
			n.hasText = true
			if n.sampleText == "" {
				n.sampleText = text
			}
			if len(n.textValues) < 60 {
				n.textValues[text]++
			}

		case xml.EndElement:
			if t.Name.Local == "flight" {
				inFlight = false
				stack = nil
				continue
			}
			if inFlight && len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func (c *census) record(n *node, attrs []xml.Attr) {
	c.seen[n]++

	for _, a := range attrs {
		if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
			continue
		}
		if a.Name.Local == "type" && a.Name.Space != "" {
			continue
		}

		at, ok := n.attrs[a.Name.Local]
		if !ok {
			at = &attr{name: a.Name.Local}
			if enumerate[a.Name.Local] {
				at.values = map[string]int{}
			}
			n.attrs[a.Name.Local] = at
		}
		if at.sample == "" {
			at.sample = a.Value
		}
		if at.values != nil && len(at.values) < 300 {
			at.values[a.Value]++
		}
		c.seenAt[at] = true
	}
}

func (c *census) flush() {
	for n, count := range c.seen {
		n.flights++
		if count > n.maxRepeat {
			n.maxRepeat = count
		}
	}
	for a := range c.seenAt {
		a.flights++
	}
	c.seen = map[*node]int{}
	c.seenAt = map[*attr]bool{}
}

// ---------- rendering ----------

func (c *census) report(w io.Writer) {
	fmt.Fprintf(w, "FIXM observed shape\n")
	fmt.Fprintf(w, "envelopes: %d    flights: %d\n\n", c.envelopes, c.flights)
	fmt.Fprintf(w, "%s\n", strings.Repeat("=", 100))
	fmt.Fprintf(w, "  pct    = share of flights carrying this element\n")
	fmt.Fprintf(w, "  >      = nesting depth below <flight>; no arrows means a direct child\n")
	fmt.Fprintf(w, "  [xN]   = seen up to N times in one flight, so it is a slice not a single value\n")
	fmt.Fprintf(w, "  @attr  = attribute of the element above it\n")
	fmt.Fprintf(w, "  *      = full value set listed in the ENUMERATED section at the end\n")
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 100))

	c.render(w, c.root, 0)

	fmt.Fprintf(w, "\n\n%s\n", strings.Repeat("=", 100))
	fmt.Fprintf(w, "ENUMERATED VALUE SETS\n")
	fmt.Fprintf(w, "%s\n", strings.Repeat("=", 100))
	c.renderEnums(w, c.root, "")
}

func (c *census) render(w io.Writer, n *node, depth int) {
	arrows := strings.Repeat(">", depth)
	indent := strings.Repeat("  ", depth)

	repeat := ""
	if n.maxRepeat > 1 {
		repeat = fmt.Sprintf(" [x%d]", n.maxRepeat)
	}

	text := ""
	if n.hasText {
		text = "  = " + truncate(n.sampleText, 40)
	}

	fmt.Fprintf(w, "%6.2f%%  %-8s %s<%s>%s%s\n",
		c.pct(n.flights), arrows, indent, n.name, repeat, text)
	var attrNames []string
	for k := range n.attrs {
		attrNames = append(attrNames, k)
	}
	sort.Slice(attrNames, func(i, j int) bool {
		return n.attrs[attrNames[i]].flights > n.attrs[attrNames[j]].flights
	})

	for _, k := range attrNames {
		a := n.attrs[k]
		marker := " "
		if a.values != nil {
			marker = "*"
		}
		fmt.Fprintf(w, "%6.2f%%  %-8s %s  %s@%-24s = %s\n",
			c.pct(a.flights), arrows, indent, marker, a.name,
			truncate(a.sample, 38))
	}

	for _, name := range n.order {
		c.render(w, n.children[name], depth+1)
	}
}

func (c *census) renderEnums(w io.Writer, n *node, path string) {
	here := path
	if here == "" {
		here = n.name
	} else {
		here = path + "/" + n.name
	}

	var names []string
	for k, a := range n.attrs {
		if len(a.values) > 0 {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	for _, k := range names {
		a := n.attrs[k]
		fmt.Fprintf(w, "\n%s/@%s   (%d distinct)\n", here, k, len(a.values))

		type kv struct {
			v string
			n int
		}
		var all []kv
		for v, cnt := range a.values {
			all = append(all, kv{v, cnt})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })

		for i, e := range all {
			if i >= 30 {
				fmt.Fprintf(w, "      ... %d more\n", len(all)-30)
				break
			}
			label := e.v
			if label == "" {
				label = "(empty)"
			}
			fmt.Fprintf(w, "      %-30s %8d\n", label, e.n)
		}
	}

	if n.hasText && len(n.textValues) > 1 && len(n.textValues) < 40 {
		fmt.Fprintf(w, "\n%s   (text, %d distinct)\n", here, len(n.textValues))
		type kv struct {
			v string
			n int
		}
		var all []kv
		for v, cnt := range n.textValues {
			all = append(all, kv{v, cnt})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
		for i, e := range all {
			if i >= 20 {
				fmt.Fprintf(w, "      ... %d more\n", len(all)-20)
				break
			}
			fmt.Fprintf(w, "      %-30s %8d\n", truncate(e.v, 30), e.n)
		}
	}

	for _, name := range n.order {
		c.renderEnums(w, n.children[name], here)
	}
}

func (c *census) pct(n int) float64 {
	if c.flights == 0 {
		return 0
	}
	return float64(n) / float64(c.flights) * 100
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ---------- main ----------

func main() {
	brokers := flag.String("brokers", envOr("KAFKA_BROKERS", "localhost:9092"), "comma-separated broker list")
	topic := flag.String("topic", envOr("KAFKA_TOPIC_RAW", "fixm.raw"), "topic to read")
	limit := flag.Int("limit", 0, "stop after N envelopes, 0 for all")
	idle := flag.Duration("idle", 5*time.Second, "stop after this long with no new messages")
	out := flag.String("out", "", "write report to this file instead of stdout")
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

	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	c := newCensus()
	log.Printf("reading %s from %s", *topic, *brokers)

	for {
		if *limit > 0 && c.envelopes >= *limit {
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

		c.envelopes++
		if err := c.walk(msg.Value); err != nil {
			log.Printf("walk failed at offset %d: %v", msg.Offset, err)
		}

		if c.envelopes%2000 == 0 {
			log.Printf("%d envelopes, %d flights", c.envelopes, c.flights)
		}
	}

	c.flush()

	w := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			log.Fatalf("create %s: %v", *out, err)
		}
		defer f.Close()
		w = f
		log.Printf("writing report to %s", *out)
	}

	c.report(w)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}