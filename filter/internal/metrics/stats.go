package metrics

import (
	"fmt"
	"time"
)

type Stats struct {
	Flights     int64
	BytesRead   int64
	ParseErrors int64
	Blocked     int64
	Forwarded   int64
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
	return fmt.Sprintf("flights=%d  %.0f/sec  forwarded=%d  blocked=%d  %d parse errors",
		s.Flights, float64(s.Flights)/elapsed, s.Forwarded, s.Blocked, s.ParseErrors)
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d flights, %.1f MB, forwarded=%d, blocked=%d, %d parse errors, %s elapsed",
		s.Flights, s.megabytes(), s.Forwarded, s.Blocked, s.ParseErrors,
		time.Since(s.start).Round(time.Millisecond))
}
