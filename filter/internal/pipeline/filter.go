package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/filter/internal/flight"
	"github.com/chasehaye/nas-pipeline/filter/internal/ladd"
	"github.com/chasehaye/nas-pipeline/filter/internal/metrics"
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
	Publish(ctx context.Context, key, data []byte) error
}

type Filter struct {
	fetcher   Fetcher
	publisher Publisher
	blocklist *ladd.Store
	workers   int
	batchSize int
}

func New(fetcher Fetcher, publisher Publisher, blocklist *ladd.Store, workers int) *Filter {
	if workers < 1 {
		workers = 1
	}
	return &Filter{
		fetcher:   fetcher,
		publisher: publisher,
		blocklist: blocklist,
		workers:   workers,
		batchSize: defaultBatchSize,
	}
}

// job = one message, tagged with the batch's WaitGroup (to signal done) and the
// batch's failure flag (set if its publish fails, to gate the commit).
type job struct {
	msg    kafka.Message
	wg     *sync.WaitGroup
	failed *atomic.Bool
}

func (f *Filter) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	// One shared channel; N persistent workers pull from it concurrently.
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
	// On exit: close the channel so workers drain & stop, then wait for them.
	defer func() {
		close(jobs)
		pool.Wait()
		log.Print(stats.Summary())
	}()

	for {
		if ok, reason := f.blocklist.Ready(); !ok {
			return fmt.Errorf("filter halted (fail-closed): %s", reason)
		}

		batch, err := f.readBatch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // shutdown: deferred cleanup runs
			}
			log.Printf("consume error: %v", err)
			continue
		}
		if len(batch) == 0 {
			continue
		}

		// Fan the batch out to the workers, then wait for ALL of it to finish.
		var wg sync.WaitGroup
		var failed atomic.Bool
		wg.Add(len(batch))
		for _, m := range batch {
			jobs <- job{msg: m, wg: &wg, failed: &failed}
		}
		wg.Wait()

		// Commit ONLY after the whole batch is published, and only if nothing
		// failed to publish (else leave it uncommitted to reprocess on restart).
		if failed.Load() {
			log.Print("batch had publish errors; not committing (will reprocess)")
			continue
		}
		// Single partition: messages arrive in offset order, so the last one is
		// the highest offset — committing it checkpoints the whole batch.
		f.commit(ctx, batch[len(batch)-1])
	}
}

// readBatch blocks for the first message, then drains up to batchSize more until
// full OR flushTimeout elapses - so light traffic never stalls waiting to fill.
func (f *Filter) readBatch(ctx context.Context) ([]kafka.Message, error) {
	first, err := f.fetcher.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	batch := make([]kafka.Message, 0, f.batchSize)
	batch = append(batch, first)

	dctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()
	for len(batch) < f.batchSize {
		msg, err := f.fetcher.Fetch(dctx)
		if err != nil {
			break // timeout or cancel: flush the partial batch
		}
		batch = append(batch, msg)
	}
	return batch, nil
}

// process = the original per-message logic, unchanged, now run by a worker.
func (f *Filter) process(ctx context.Context, msg kafka.Message, stats *metrics.Stats, failed *atomic.Bool) {
	n := stats.Flights.Add(1)
	stats.BytesRead.Add(int64(len(msg.Value)))
	if n%1000 == 0 {
		log.Print(stats.Progress())
	}

	m, err := flight.Parse(msg.Value)
	if err != nil {
		stats.ParseErrors.Add(1)
		log.Printf("parse error offset %d (dropped): %v", msg.Offset, err)
		return // dropped, but legitimately "done"
	}

	if f.blocklist.Blocks(m.Ident.CallSign, m.Ident.Registration) {
		stats.Blocked.Add(1)
		return // intentionally not published
	}

	if err := f.publisher.Publish(ctx, []byte(m.Gufi), m.Raw); err != nil {
		log.Printf("publish error offset %d: %v", msg.Offset, err)
		failed.Store(true) // gate the batch commit
		return
	}
	stats.Forwarded.Add(1)
}

func (f *Filter) commit(ctx context.Context, msg kafka.Message) {
	if err := f.fetcher.Commit(ctx, msg); err != nil {
		log.Printf("commit error: %v", err)
	}
}
