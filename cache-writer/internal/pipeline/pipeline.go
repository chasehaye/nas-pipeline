package pipeline

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"

	"github.com/chasehaye/nas-pipeline/redis-service/internal/flight"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/metrics"
)

type Fetcher interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Writer interface {
	UpsertFlight(ctx context.Context, f flight.Flight) error
	DeleteFlight(ctx context.Context, gufi string) error
}

type DLQPublisher interface {
	Publish(ctx context.Context, orig kafka.Message, stage, class, cause string) error
}

const statusActive = "ACTIVE"

type Pipeline struct {
	fetcher Fetcher
	writer  Writer
	dlq     DLQPublisher
}

func New(fetcher Fetcher, writer Writer, dlq DLQPublisher) *Pipeline {
	return &Pipeline{fetcher: fetcher, writer: writer, dlq: dlq}
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	for {
		msg, err := p.fetcher.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("shutting down", "stats", stats)
				return nil
			}
			slog.Error("consume error", "err", err)
			continue
		}
		p.process(ctx, msg, stats)
	}
}

func (p *Pipeline) process(ctx context.Context, msg kafka.Message, stats *metrics.Stats) {
	start := time.Now()
	defer func() { metrics.ProcessDuration.Observe(time.Since(start).Seconds()) }()

	stats.Messages++
	metrics.Messages.Inc()
	stats.BytesRead += int64(len(msg.Value))
	if stats.Messages%1000 == 0 {
		slog.Info("progress", "stats", stats)
	}

	// Parse failure is poison → dead-letter, then commit.
	f, err := flight.Parse(msg.Value)
	if err != nil {
		stats.ParseErrors++
		metrics.ParseErrors.Inc()
		if derr := p.dlq.Publish(ctx, msg, "cache-writer", "poison", err.Error()); derr != nil {
			slog.Error("dlq publish failed", "offset", msg.Offset, "err", derr)
			return // DLQ write failed (transient) — don't commit, reprocess
		}
		metrics.DLQPublished.Inc()
		slog.Warn("dead-lettered poison message", "offset", msg.Offset, "err", err)
		p.commit(ctx, msg)
		return
	}

	// Non-active flight: drop from the cache. Redis errors are transient → retry.
	if f.Status != statusActive {
		if err := kafkax.Do(ctx, kafkax.DefaultPolicy, func() error {
			return p.writer.DeleteFlight(ctx, f.Gufi)
		}); err != nil {
			metrics.WriteErrors.Inc()
			slog.Error("redis delete failed after retries", "gufi", f.Gufi, "err", err)
			return
		}
		stats.Removed++
		metrics.Removed.Inc()
		p.commit(ctx, msg)
		return
	}

	// Active but no position: nothing to store.
	if !f.HasPosition {
		stats.NoPosition++
		metrics.NoPosition.Inc()
		p.commit(ctx, msg)
		return
	}

	// Active with a position: upsert. Redis errors are transient → retry.
	if err := kafkax.Do(ctx, kafkax.DefaultPolicy, func() error {
		return p.writer.UpsertFlight(ctx, f)
	}); err != nil {
		metrics.WriteErrors.Inc()
		slog.Error("redis upsert failed after retries", "gufi", f.Gufi, "err", err)
		return
	}
	stats.Stored++
	metrics.Stored.Inc()
	p.commit(ctx, msg)
}

func (p *Pipeline) commit(ctx context.Context, msg kafka.Message) {
	if err := p.fetcher.Commit(ctx, msg); err != nil {
		slog.Error("commit failed", "err", err)
	}
}
