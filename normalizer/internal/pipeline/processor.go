package pipeline

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

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

type Processor struct {
	fetcher   Fetcher
	publisher Publisher
	workers   int
	batchSize int
}

func New(fetcher Fetcher, publisher Publisher, workers int) *Processor {
	if workers < 1 {
		workers = 1
	}
	return &Processor{
		fetcher:   fetcher,
		publisher: publisher,
		workers:   workers,
		batchSize: defaultBatchSize,
	}
}

// job = one raw envelope to process, tagged with the batch's WaitGroup (to
// signal done) and the batch's failure flag (set if its flights fail to
// publish, to gate the commit).
type job struct {
	msg    kafka.Message
	wg     *sync.WaitGroup
	failed *atomic.Bool
}

func (p *Processor) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	// One shared channel; N persistent workers pull from it concurrently. The
	// per-envelope work (XML parse -> encode -> publish) is CPU-heavy
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
		log.Print(stats.Summary())
	}()

	for {
		batch, err := p.readBatch(ctx)
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
		// failed (else leave it uncommitted to reprocess on restart).
		if failed.Load() {
			log.Print("batch had publish errors; not committing (will reprocess)")
			continue
		}
		// Single partition: messages arrive in offset order, so the last one is
		// the highest offset — committing it checkpoints the whole batch.
		p.commit(ctx, batch[len(batch)-1])
	}
}

// readBatch blocks for the first envelope, then drains up to batchSize more
// until full OR flushTimeout elapses — so light traffic never stalls.
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
			break // timeout or cancel: flush the partial batch
		}
		batch = append(batch, msg)
	}
	return batch, nil
}

// process = parse one envelope into flights and publish them (original logic,
// now run by a worker). One envelope in -> many flights out.
func (p *Processor) process(ctx context.Context, msg kafka.Message, stats *metrics.Stats, failed *atomic.Bool) {
	n := stats.Envelopes.Add(1)
	stats.BytesRead.Add(int64(len(msg.Value)))
	if n%1000 == 0 {
		log.Print(stats.Progress())
	}

	flights, err := fixm.ParseEnvelope(msg.Value)
	if err != nil {
		stats.ParseErrors.Add(1)
		log.Printf("process error offset %d: %v", msg.Offset, err)
		return // dropped, but legitimately "done"
	}

	if err := p.publishFlights(ctx, flights); err != nil {
		log.Printf("publish error offset %d: %v", msg.Offset, err)
		failed.Store(true) // gate the batch commit
		return
	}
}

func (p *Processor) publishFlights(ctx context.Context, flights []fixm.Message) error {
	if len(flights) == 0 {
		return nil
	}

	msgs := make([]kafka.Message, 0, len(flights))
	for _, f := range flights {
		data, err := fixm.EncodeOne(f)
		if err != nil {
			return err
		}
		msgs = append(msgs, kafka.Message{
			Key:   []byte(f.Flight.Gufi.Code),
			Value: data,
		})
	}

	return p.publisher.Publish(ctx, msgs...)
}

func (p *Processor) commit(ctx context.Context, msg kafka.Message) {
	if err := p.fetcher.Commit(ctx, msg); err != nil {
		log.Printf("commit error: %v", err)
	}
}
