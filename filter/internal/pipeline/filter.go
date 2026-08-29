package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"

	"github.com/chasehaye/nas-pipeline/filter/internal/flight"
	"github.com/chasehaye/nas-pipeline/filter/internal/ladd"
	"github.com/chasehaye/nas-pipeline/filter/internal/metrics"
)

const (
	defaultBatchSize = 100
	flushTimeout     = 200 * time.Millisecond
)

type Consumer interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Publisher interface {
	Publish(ctx context.Context, key, data []byte) error
}

type DLQPublisher interface {
	Publish(ctx context.Context, orig kafka.Message, stage, class, cause string) error
}

type Filter struct {
	consumer  Consumer
	publisher Publisher
	dlq       DLQPublisher
	blocklist *ladd.Store
	workers   int
	batchSize int
}

func New(consumer Consumer, publisher Publisher, dlq DLQPublisher, blocklist *ladd.Store, workers int) *Filter {
	if workers < 1 {
		workers = 1
	}
	return &Filter{
		consumer:  consumer,
		publisher: publisher,
		dlq:       dlq,
		blocklist: blocklist,
		workers:   workers,
		batchSize: defaultBatchSize,
	}
}

// job = one message tagged with its batch's WaitGroup and shared failure flag.
type job struct {
	msg    kafka.Message
	wg     *sync.WaitGroup
	failed *atomic.Bool
}

func (f *Filter) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	jobs := make(chan job, f.batchSize)
	var pool sync.WaitGroup
	for i := 0; i < f.workers; i++ {
		pool.Add(1)
		go func() {
			defer pool.Done()
			for j := range jobs {
				f.process(ctx, j.msg, stats, j.failed)
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
		// Fail closed: refuse to forward traffic if the LADD list is missing/stale.
		if ok, reason := f.blocklist.Ready(); !ok {
			return fmt.Errorf("filter halted (fail-closed): %s", reason)
		}

		batch, err := f.readBatch(ctx)
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
		f.commit(ctx, batch[len(batch)-1])
	}
}

// readBatch blocks for the first message, then drains up to batchSize more until
// full or flushTimeout elapses.
func (f *Filter) readBatch(ctx context.Context) ([]kafka.Message, error) {
	first, err := f.consumer.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	batch := make([]kafka.Message, 0, f.batchSize)
	batch = append(batch, first)

	dctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()
	for len(batch) < f.batchSize {
		msg, err := f.consumer.Fetch(dctx)
		if err != nil {
			break
		}
		batch = append(batch, msg)
	}
	return batch, nil
}

func (f *Filter) process(ctx context.Context, msg kafka.Message, stats *metrics.Stats, failed *atomic.Bool) {
	start := time.Now()
	defer func() { metrics.ProcessDuration.Observe(time.Since(start).Seconds()) }()

	n := stats.Flights.Add(1)
	metrics.FlightsProcessed.Inc()
	stats.BytesRead.Add(int64(len(msg.Value)))
	if n%1000 == 0 {
		slog.Info("progress", "stats", stats)
	}

	// Parse failure is poison → dead-letter, don't retry.
	m, err := flight.Parse(msg.Value)
	if err != nil {
		stats.ParseErrors.Add(1)
		metrics.ParseErrors.Inc()
		f.deadLetter(ctx, msg, "parse", err, failed)
		return
	}

	// A blocked flight is intentionally suppressed, not an error.
	if f.blocklist.Blocks(m.Ident.CallSign, m.Ident.Registration) {
		stats.Blocked.Add(1)
		metrics.Blocked.Inc()
		return
	}

	// Publish failure is transient → retry; if still failing, gate the commit.
	err = kafkax.Do(ctx, kafkax.DefaultPolicy, func() error {
		return f.publisher.Publish(ctx, []byte(m.Gufi), m.Raw)
	})
	if err != nil {
		metrics.PublishErrors.Inc()
		slog.Error("publish failed after retries", "offset", msg.Offset, "err", err)
		failed.Store(true)
		return
	}
	stats.Forwarded.Add(1)
	metrics.Forwarded.Inc()
}

// deadLetter quarantines a poison message. A failed DLQ write is transient, so
// gate the commit rather than lose the message.
func (f *Filter) deadLetter(ctx context.Context, msg kafka.Message, reason string, cause error, failed *atomic.Bool) {
	if err := f.dlq.Publish(ctx, msg, "filter", "poison", cause.Error()); err != nil {
		slog.Error("dlq publish failed", "offset", msg.Offset, "reason", reason, "err", err)
		failed.Store(true)
		return
	}
	metrics.DLQPublished.Inc()
	slog.Warn("dead-lettered poison message", "offset", msg.Offset, "reason", reason, "err", cause)
}

func (f *Filter) commit(ctx context.Context, msg kafka.Message) {
	if err := f.consumer.Commit(ctx, msg); err != nil {
		slog.Error("commit failed", "err", err)
	}
}
