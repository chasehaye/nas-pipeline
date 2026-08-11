package metrics

import (
	"fmt"
	"time"
)

type Stats struct {
	Messages    int64
	BytesRead   int64
	ParseErrors int64
	NoPosition  int64 // dropped: no usable enRoute position (not airborne yet)
	Stored      int64
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
	return fmt.Sprintf("messages=%d  %.0f/sec  stored=%d  no-position=%d  %d parse errors",
		s.Messages, float64(s.Messages)/elapsed, s.Stored, s.NoPosition, s.ParseErrors)
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d messages, %.1f MB, stored=%d, no-position=%d, %d parse errors, %s elapsed",
		s.Messages, s.megabytes(), s.Stored, s.NoPosition, s.ParseErrors,
		time.Since(s.start).Round(time.Millisecond))
}
