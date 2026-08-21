package metrics

import (
	"fmt"
	"time"
)

type Stats struct {
	Messages    int64
	BytesRead   int64
	ParseErrors int64
	Skipped     int64
	Recorded    int64
	Positions   int64
	start       time.Time
}

func NewStats() *Stats { return &Stats{start: time.Now()} }

func (s *Stats) megabytes() float64 { return float64(s.BytesRead) / 1024 / 1024 }

func (s *Stats) Progress() string {
	elapsed := time.Since(s.start).Seconds()
	return fmt.Sprintf("messages=%d  %.0f/sec  recorded=%d  positions=%d  skipped=%d  %d parse errors",
		s.Messages, float64(s.Messages)/elapsed, s.Recorded, s.Positions, s.Skipped, s.ParseErrors)
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d messages, %.1f MB, recorded=%d, positions=%d, skipped=%d, %d parse errors, %s elapsed",
		s.Messages, s.megabytes(), s.Recorded, s.Positions, s.Skipped, s.ParseErrors,
		time.Since(s.start).Round(time.Millisecond))
}
