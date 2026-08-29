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

	"github.com/chasehaye/nas-pipeline/database-writer/internal/flight"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/metrics"
)

type Consumer interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Writer interface {
	RecordBatch(ctx context.Context, flights []flight.Flight) error
}

type DLQPublisher interface {
	Publish(ctx context.Context, orig kafka.Message, stage, class, cause string) error
}

type Pipeline struct {
	consumer     Consumer
	writer       Writer
	dlq          DLQPublisher
	batchSize    int
	flushTimeout time.Duration
}

func New(consumer Consumer, writer Writer, dlq DLQPublisher, batchSize int, flushTimeout time.Duration) *Pipeline {
	return &Pipeline{
		consumer:     consumer,
		writer:       writer,
		dlq:          dlq,
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
	}
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()
	var nextLog int64 = 5000

	for {
		batch := p.readBatch(ctx)
		if len(batch) == 0 {
			if ctx.Err() != nil {
				slog.Info("shutting down", "stats", stats)
				return nil
			}
			continue
		}

		flights := make([]flight.Flight, 0, len(batch))
		for _, msg := range batch {
			stats.Messages++
			metrics.Messages.Inc()
			stats.BytesRead += int64(len(msg.Value))

			f, err := flight.Parse(msg.Value)
			if err != nil {
				// Poison — dead-letter before the batch commit sweeps past it.
				stats.ParseErrors++
				metrics.ParseErrors.Inc()
				p.deadLetter(ctx, msg, err)
				continue
			}
			if f.Gufi == "" {
				stats.Skipped++
				metrics.Skipped.Inc()
				continue
			}
			flights = append(flights, f)
		}

		// Durable history: retry the batch write until it succeeds (backpressure),
		// not bounded-retry + DLQ, so a brief Postgres outage never drops good data.
		start := time.Now()
		for {
			if err := p.writer.RecordBatch(ctx, flights); err != nil {
				if ctx.Err() != nil {
					slog.Info("shutting down", "stats", stats)
					return nil
				}
				metrics.BatchWriteErrors.Inc()
				slog.Error("batch write failed (retrying)", "err", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
		metrics.BatchDuration.Observe(time.Since(start).Seconds())

		for _, f := range flights {
			stats.Recorded++
			metrics.Recorded.Inc()
			if f.HasPosition {
				stats.Positions++
				metrics.Positions.Inc()
			}
		}

		if err := p.consumer.Commit(ctx, batch[len(batch)-1]); err != nil {
			slog.Error("commit failed", "err", err)
		}

		if stats.Messages >= nextLog {
			slog.Info("progress", "stats", stats)
			nextLog += 5000
		}
	}
}

// deadLetter best-effort quarantines a poison message; the batch commit proceeds
// regardless, so a DLQ write failure is logged rather than allowed to stall history.
func (p *Pipeline) deadLetter(ctx context.Context, msg kafka.Message, cause error) {
	err := kafkax.Do(ctx, kafkax.DefaultPolicy, func() error {
		return p.dlq.Publish(ctx, msg, "database-writer", "poison", cause.Error())
	})
	if err != nil {
		slog.Error("dlq publish failed (message may be lost on commit)", "offset", msg.Offset, "err", err)
		return
	}
	metrics.DLQPublished.Inc()
	slog.Warn("dead-lettered poison message", "offset", msg.Offset, "err", cause)
}

// readBatch blocks for the first message, then fills up to batchSize until the
// flush window elapses.
func (p *Pipeline) readBatch(ctx context.Context) []kafka.Message {
	batch := make([]kafka.Message, 0, p.batchSize)

	msg, err := p.consumer.Fetch(ctx)
	if err != nil {
		return batch
	}
	batch = append(batch, msg)

	deadline, cancel := context.WithTimeout(ctx, p.flushTimeout)
	defer cancel()
	for len(batch) < p.batchSize {
		msg, err := p.consumer.Fetch(deadline)
		if err != nil {
			break
		}
		batch = append(batch, msg)
	}
	return batch
}
