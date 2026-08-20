package metrics

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Stats struct {
	Envelopes   atomic.Int64
	BytesRead   atomic.Int64
	ParseErrors atomic.Int64
	start       time.Time
}

func NewStats() *Stats {
	return &Stats{start: time.Now()}
}

func (s *Stats) megabytes() float64 {
	return float64(s.BytesRead.Load()) / 1024 / 1024
}

func (s *Stats) Progress() string {
	elapsed := time.Since(s.start).Seconds()
	envelopes := s.Envelopes.Load()
	return fmt.Sprintf("envelopes=%d  %.1f MB  %.0f/sec  %d parse errors",
		envelopes, s.megabytes(), float64(envelopes)/elapsed, s.ParseErrors.Load())
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d envelopes, %.1f MB, %d parse errors, %s elapsed",
		s.Envelopes.Load(), s.megabytes(), s.ParseErrors.Load(),
		time.Since(s.start).Round(time.Millisecond))
}
