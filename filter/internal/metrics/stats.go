package metrics

import (
	"log/slog"
	"sync/atomic"
	"time"
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

// LogValue implements slog.LogValuer so a *Stats logs as a structured group.
func (s *Stats) LogValue() slog.Value {
	elapsed := time.Since(s.start)
	secs := elapsed.Seconds()
	flights := s.Flights.Load()

	perSec := 0.0
	if secs > 0 {
		perSec = float64(flights) / secs
	}

	return slog.GroupValue(
		slog.Int64("flights", flights),
		slog.Float64("mb", s.megabytes()),
		slog.Float64("per_sec", perSec),
		slog.Int64("forwarded", s.Forwarded.Load()),
		slog.Int64("blocked", s.Blocked.Load()),
		slog.Int64("parse_errors", s.ParseErrors.Load()),
		slog.Duration("elapsed", elapsed.Round(time.Millisecond)),
	)
}
