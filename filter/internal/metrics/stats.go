package metrics

import (
	"fmt"
	"time"
)

type Stats struct {
	Envelopes   int64 // normalized messages consumed
	BytesRead   int64
	ParseErrors int64 // messages dropped because they could not be parsed
	Blocked     int64 // individual flights removed by the LADD screen
	Forwarded   int64 // envelopes published to the filtered topic
	Dropped     int64 // envelopes with no surviving flights
	start       time.Time
}

func NewStats() *Stats {
	return &Stats{start: time.Now()}
}

func (s *Stats) megabytes() float64 {
	return float64(s.BytesRead) / 1024 / 1024
}

func (s *Stats) Progress() string {
	elapsed := time.Since(s.start).Seconds()
	return fmt.Sprintf("envelopes=%d  %.0f/sec  forwarded=%d  blocked=%d  dropped=%d  %d parse errors",
		s.Envelopes, float64(s.Envelopes)/elapsed, s.Forwarded, s.Blocked, s.Dropped, s.ParseErrors)
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d envelopes, %.1f MB, forwarded=%d, blocked=%d, dropped=%d, %d parse errors, %s elapsed",
		s.Envelopes, s.megabytes(), s.Forwarded, s.Blocked, s.Dropped, s.ParseErrors,
		time.Since(s.start).Round(time.Millisecond))
}
