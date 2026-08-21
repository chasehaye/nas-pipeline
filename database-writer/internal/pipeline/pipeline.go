package pipeline

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/database-writer/internal/flight"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/metrics"
)

type Fetcher interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

type Writer interface {
	RecordBatch(ctx context.Context, flights []flight.Flight) error
}

type Pipeline struct {
	fetcher      Fetcher
	writer       Writer
	batchSize    int
	flushTimeout time.Duration
}

func New(fetcher Fetcher, writer Writer, batchSize int, flushTimeout time.Duration) *Pipeline {
	return &Pipeline{
		fetcher:      fetcher,
		writer:       writer,
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
				log.Print(stats.Summary())
				return nil
			}
			continue
		}

		flights := make([]flight.Flight, 0, len(batch))
		for _, msg := range batch {
			stats.Messages++
			stats.BytesRead += int64(len(msg.Value))

			f, err := flight.Parse(msg.Value)
			if err != nil {
				stats.ParseErrors++
				log.Printf("parse error offset %d (dropped): %v", msg.Offset, err)
				continue
			}
			if f.Gufi == "" {
				stats.Skipped++
				continue
			}
			flights = append(flights, f)
		}

		// One transaction per batch. Retry the whole batch on failure so the
		// offset is only committed once the write is durable
		for {
			if err := p.writer.RecordBatch(ctx, flights); err != nil {
				if ctx.Err() != nil {
					log.Print(stats.Summary())
					return nil
				}
				log.Printf("batch write error (retrying): %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for _, f := range flights {
			stats.Recorded++
			if f.HasPosition {
				stats.Positions++
			}
		}

		// Committing the last offset commits everything earlier in the batch.
		if err := p.fetcher.Commit(ctx, batch[len(batch)-1]); err != nil {
			log.Printf("commit error: %v", err)
		}

		if stats.Messages >= nextLog {
			log.Print(stats.Progress())
			nextLog += 5000
		}
	}
}

// readBatch blocks for the first message, then fills up to batchSize until the
// flush window (measured from that first message) elapses.
func (p *Pipeline) readBatch(ctx context.Context) []kafka.Message {
	batch := make([]kafka.Message, 0, p.batchSize)

	msg, err := p.fetcher.Fetch(ctx)
	if err != nil {
		return batch
	}
	batch = append(batch, msg)

	deadline, cancel := context.WithTimeout(ctx, p.flushTimeout)
	defer cancel()
	for len(batch) < p.batchSize {
		msg, err := p.fetcher.Fetch(deadline)
		if err != nil {
			break
		}
		batch = append(batch, msg)
	}
	return batch
}
