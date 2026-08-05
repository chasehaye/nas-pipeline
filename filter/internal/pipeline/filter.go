package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/filter/internal/flight"
	"github.com/chasehaye/nas-pipeline/filter/internal/ladd"
	"github.com/chasehaye/nas-pipeline/filter/internal/metrics"
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
}

func New(fetcher Fetcher, publisher Publisher, blocklist *ladd.Store) *Filter {
	return &Filter{
		fetcher:   fetcher,
		publisher: publisher,
		blocklist: blocklist,
	}
}

func (f *Filter) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := metrics.NewStats()

	for {
		if ok, reason := f.blocklist.Ready(); !ok {
			return fmt.Errorf("filter halted (fail-closed): %s", reason)
		}

		msg, err := f.fetcher.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Print(stats.Summary())
				return nil
			}
			log.Printf("consume error: %v", err)
			continue
		}

		stats.Flights++
		stats.BytesRead += int64(len(msg.Value))
		if stats.Flights%1000 == 0 {
			log.Print(stats.Progress())
		}

		m, err := flight.Parse(msg.Value)
		if err != nil {
			stats.ParseErrors++
			log.Printf("parse error offset %d (dropped): %v", msg.Offset, err)
			f.commit(ctx, msg)
			continue
		}

		if f.blocklist.Blocks(m.Ident.CallSign, m.Ident.Registration) {
			stats.Blocked++
			f.commit(ctx, msg)
			continue
		}

		if err := f.publisher.Publish(ctx, []byte(m.Gufi), m.Raw); err != nil {
			log.Printf("publish error: %v", err)
			continue
		}
		stats.Forwarded++

		f.commit(ctx, msg)
	}
}

func (f *Filter) commit(ctx context.Context, msg kafka.Message) {
	if err := f.fetcher.Commit(ctx, msg); err != nil {
		log.Printf("commit error: %v", err)
	}
}
