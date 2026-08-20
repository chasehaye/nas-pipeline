package metrics

import (
	"fmt"
	"time"
	"sync/atomic" 
)

type Stats struct {
	Flights     atomic.Int64
	BytesRead   atomic.Int64
	ParseErrors atomic.Int64
	Blocked     atomic.Int64
	Forwarded   atomic.Int64
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
	flights := s.Flights.Load()
	return fmt.Sprintf("flights=%d  %.0f/sec  forwarded=%d  blocked=%d  %d parse errors",
		flights, float64(flights)/elapsed, s.Forwarded.Load(), s.Blocked.Load(), s.ParseErrors.Load())
}

func (s *Stats) Summary() string {
	return fmt.Sprintf("stopped: %d flights, %.1f MB, forwarded=%d, blocked=%d, %d parse errors, %s elapsed",
		s.Flights.Load(), s.megabytes(), s.Forwarded.Load(), s.Blocked.Load(), s.ParseErrors.Load(),
		time.Since(s.start).Round(time.Millisecond))
}