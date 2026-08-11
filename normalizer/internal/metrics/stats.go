package metrics

import (
	"fmt"
	"time"
)


type Stats struct {
	Envelopes   int64
	BytesRead   int64
	ParseErrors int64
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
	return fmt.Sprintf("envelopes=%d  %.1f MB  %.0f/sec  %d parse errors",
		s.Envelopes, s.megabytes(), float64(s.Envelopes)/elapsed, s.ParseErrors)
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d envelopes, %.1f MB, %d parse errors, %s elapsed",
		s.Envelopes, s.megabytes(), s.ParseErrors,
		time.Since(s.start).Round(time.Millisecond))
}