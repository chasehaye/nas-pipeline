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

// Fetcher is anything we can pull messages from and commit back to.
// *kafka.Consumer satisfies this as-is.
type Fetcher interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, msg kafka.Message) error
}

// Publisher is anything we can publish normalized bytes to.
// *kafka.Producer satisfies this as-is.
type Publisher interface {
	Publish(ctx context.Context, data []byte) error
}

type Processor struct {
	fetcher   Fetcher
	publisher Publisher
}

func New(fetcher Fetcher, publisher Publisher) *Processor {
	return &Processor{fetcher: fetcher, publisher: publisher}
}

// Run consumes, processes, publishes, and commits in a loop until the context
// is cancelled or an interrupt/SIGTERM is received.
func (p *Processor) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	for {
		msg, err := p.fetcher.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Print(stats.Summary())
				return nil // graceful shutdown
			}
			log.Printf("consume error: %v", err)
			continue
		}

		stats.Envelopes++
		stats.BytesRead += int64(len(msg.Value))
		if stats.Envelopes%1000 == 0 {
			log.Print(stats.Progress())
		}

		data, err := Process(msg.Value)
		if err != nil {
			stats.ParseErrors++
			log.Printf("process error offset %d: %v", msg.Offset, err)
			continue
		}

		if err := p.publisher.Publish(ctx, data); err != nil {
			log.Printf("publish error: %v", err)
			continue
		}

		if err := p.fetcher.Commit(ctx, msg); err != nil {
			log.Printf("commit error: %v", err)
		}
	}
}

// Process is the pure transform: parse a FIXM envelope and encode it as JSON.
func Process(data []byte) ([]byte, error) {
	messages, err := fixm.ParseEnvelope(data)
	if err != nil {
		return nil, err
	}

	return fixm.EncodeJSON(messages)
}
