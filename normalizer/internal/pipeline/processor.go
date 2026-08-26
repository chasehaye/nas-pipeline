package pipeline

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"

	"github.com/chasehaye/nas-pipeline/processor/internal/fixm"
	"github.com/chasehaye/nas-pipeline/processor/internal/metrics"
)

const (
	defaultBatchSize = 100
	flushTimeout     = 200 * time.Millisecond
)

type Fetcher interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Publisher interface {
	Publish(ctx context.Context, msgs ...kafka.Message) error
}

type DLQPublisher interface {
	Publish(ctx context.Context, orig kafka.Message, stage, class, cause string) error
}

type Processor struct {
	fetcher   Fetcher
	publisher Publisher
	dlq       DLQPublisher
	workers   int
	batchSize int
}

func New(fetcher Fetcher, publisher Publisher, dlq DLQPublisher, workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{
		fetcher:   fetcher,
		publisher: publisher,
		dlq:       dlq,
		workers:   workers,
		batchSize: defaultBatchSize,
	}
}

// job = one envelope tagged with its batch's WaitGroup and shared failure flag.
type job struct {
	msg    kafka.Message
	wg     *sync.WaitGroup
	failed *atomic.Bool
}

func (p *Processor) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	jobs := make(chan job, p.batchSize)
	var pool sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		pool.Add(1)
		go func() {
			defer pool.Done()
			for j := range jobs {
				p.process(ctx, j.msg, stats, j.failed)
				j.wg.Done()
			}
		}()
	}
	defer func() {
		close(jobs)
		pool.Wait()
		slog.Info("shutting down", "stats", stats)
	}()

	for {
		batch, err := p.readBatch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("consume error", "err", err)
			continue
		}
		if len(batch) == 0 {
			continue
		}

		var wg sync.WaitGroup
		var failed atomic.Bool
		wg.Add(len(batch))
		for _, m := range batch {
			jobs <- job{msg: m, wg: &wg, failed: &failed}
		}
		wg.Wait()

		// Commit the batch only if nothing failed; otherwise reprocess on restart.
		if failed.Load() {
			slog.Warn("batch had publish errors; not committing (will reprocess)")
			continue
		}
		p.commit(ctx, batch[len(batch)-1])
	}
}

// readBatch blocks for the first envelope, then drains up to batchSize more
// until full or flushTimeout elapses.
func (p *Processor) readBatch(ctx context.Context) ([]kafka.Message, error) {
	first, err := p.fetcher.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	batch := make([]kafka.Message, 0, p.batchSize)
	batch = append(batch, first)

	dctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()
	for len(batch) < p.batchSize {
		msg, err := p.fetcher.Fetch(dctx)
		if err != nil {
			break
		}
		batch = append(batch, msg)
	}
	return batch, nil
}

func (p *Processor) process(ctx context.Context, msg kafka.Message, stats *metrics.Stats, failed *atomic.Bool) {
	start := time.Now()
	defer func() { metrics.ProcessDuration.Observe(time.Since(start).Seconds()) }()

	n := stats.Envelopes.Add(1)
	metrics.EnvelopesProcessed.Inc()
	stats.BytesRead.Add(int64(len(msg.Value)))
	if n%1000 == 0 {
		slog.Info("progress", "stats", stats)
	}

	// Parse / encode failures are poison → dead-letter, don't retry.
	flights, err := fixm.ParseEnvelope(msg.Value)
	if err != nil {
		stats.ParseErrors.Add(1)
		metrics.ParseErrors.Inc()
		p.deadLetter(ctx, msg, "parse", err, failed)
		return
	}

	msgs, err := encodeFlights(flights)
	if err != nil {
		p.deadLetter(ctx, msg, "encode", err, failed)
		return
	}
	if len(msgs) == 0 {
		return
	}

	// Publish failure is transient → retry; if still failing, gate the commit.
	err = kafkax.Do(ctx, kafkax.DefaultPolicy, func() error {
		return p.publisher.Publish(ctx, msgs...)
	})
	if err != nil {
		metrics.PublishErrors.Inc()
		slog.Error("publish failed after retries", "offset", msg.Offset, "err", err)
		failed.Store(true)
		return
	}

	metrics.FlightsPublished.Add(float64(len(msgs)))
}

// deadLetter quarantines a poison message. A failed DLQ write is transient, so
// gate the commit rather than lose the message.
func (p *Processor) deadLetter(ctx context.Context, msg kafka.Message, reason string, cause error, failed *atomic.Bool) {
	if err := p.dlq.Publish(ctx, msg, "normalizer", "poison", cause.Error()); err != nil {
		slog.Error("dlq publish failed", "offset", msg.Offset, "reason", reason, "err", err)
		failed.Store(true)
		return
	}
	metrics.DLQPublished.Inc()
	slog.Warn("dead-lettered poison envelope", "offset", msg.Offset, "reason", reason, "err", cause)
}

func encodeFlights(flights []fixm.Message) ([]kafka.Message, error) {
	if len(flights) == 0 {
		return nil, nil
	}

	msgs := make([]kafka.Message, 0, len(flights))
	for _, f := range flights {
		data, err := fixm.EncodeOne(f)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, kafka.Message{
			Key:   []byte(f.Flight.Gufi.Code),
			Value: data,
		})
	}

	return msgs, nil
}

func (p *Processor) commit(ctx context.Context, msg kafka.Message) {
	if err := p.fetcher.Commit(ctx, msg); err != nil {
		slog.Error("commit failed", "err", err)
	}
}
