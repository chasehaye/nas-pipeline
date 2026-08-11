package pipeline

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

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

const statusActive = "ACTIVE"

type Pipeline struct {
	fetcher Fetcher
	writer  Writer
}

func New(fetcher Fetcher, writer Writer) *Pipeline {
	return &Pipeline{fetcher: fetcher, writer: writer}
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	for {
		msg, err := p.fetcher.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Print(stats.Summary())
				return nil
			}
			log.Printf("consume error: %v", err)
			continue
		}

		stats.Messages++
		stats.BytesRead += int64(len(msg.Value))
		if stats.Messages%1000 == 0 {
			log.Print(stats.Progress())
		}

		f, err := flight.Parse(msg.Value)
		if err != nil {
			stats.ParseErrors++
			log.Printf("parse error offset %d (dropped): %v", msg.Offset, err)
			p.commit(ctx, msg)
			continue
		}

		if f.Status != statusActive {
			if err := p.writer.DeleteFlight(ctx, f.Gufi); err != nil {
				log.Printf("redis delete error (will retry): %v", err)
				continue
			}
			stats.Removed++
			p.commit(ctx, msg)
			continue
		}

		if !f.HasPosition {
			stats.NoPosition++
			p.commit(ctx, msg)
			continue
		}

		if err := p.writer.UpsertFlight(ctx, f); err != nil {
			log.Printf("redis upsert error (will retry): %v", err)
			continue
		}
		stats.Stored++

		p.commit(ctx, msg)
	}
}

func (p *Pipeline) commit(ctx context.Context, msg kafka.Message) {
	if err := p.fetcher.Commit(ctx, msg); err != nil {
		log.Printf("commit error: %v", err)
	}
}
