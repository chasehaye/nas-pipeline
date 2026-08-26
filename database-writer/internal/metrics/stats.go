package metrics

import (
	"log/slog"
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

// LogValue implements slog.LogValuer so a *Stats logs as a structured group.
func (s *Stats) LogValue() slog.Value {
	elapsed := time.Since(s.start)
	secs := elapsed.Seconds()

	perSec := 0.0
	if secs > 0 {
		perSec = float64(s.Messages) / secs
	}

	return slog.GroupValue(
		slog.Int64("messages", s.Messages),
		slog.Float64("mb", s.megabytes()),
		slog.Float64("per_sec", perSec),
		slog.Int64("recorded", s.Recorded),
		slog.Int64("positions", s.Positions),
		slog.Int64("skipped", s.Skipped),
		slog.Int64("parse_errors", s.ParseErrors),
		slog.Duration("elapsed", elapsed.Round(time.Millisecond)),
	)
}
