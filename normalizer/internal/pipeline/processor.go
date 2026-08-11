package pipeline

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/processor/internal/fixm"
	"github.com/chasehaye/nas-pipeline/processor/internal/metrics"
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
}

func New(fetcher Fetcher, publisher Publisher) *Processor {
	return &Processor{fetcher: fetcher, publisher: publisher}
}


func (p *Processor) Run(ctx context.Context) error {
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

		stats.Envelopes++
		stats.BytesRead += int64(len(msg.Value))
		if stats.Envelopes%1000 == 0 {
			log.Print(stats.Progress())
		}

		flights, err := fixm.ParseEnvelope(msg.Value)
		if err != nil {
			stats.ParseErrors++
			log.Printf("process error offset %d: %v", msg.Offset, err)
			continue
		}

		if err := p.publishFlights(ctx, flights); err != nil {
			log.Printf("publish error offset %d (will retry envelope): %v", msg.Offset, err)
			continue
		}

		if err := p.fetcher.Commit(ctx, msg); err != nil {
			log.Printf("commit error: %v", err)
		}
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
