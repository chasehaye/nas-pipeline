// Package consumer owns the Kafka read loop and its commit semantics.
package consumer

import (
	"context"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"

	"github.com/chasehaye/nas-pipeline/processor/internal/fixm"
)

type Config struct {
	Brokers string
	Topic   string
	Group   string
}

type Consumer struct {
	reader *kafka.Reader
}

func New(cfg Config) *Consumer {
	log.Printf("consuming %s from %s as group %q",
		cfg.Topic, cfg.Brokers, cfg.Group)

	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: strings.Split(cfg.Brokers, ","),
			Topic:   cfg.Topic,
			GroupID:  cfg.Group,
			MinBytes: 1,
			MaxBytes: 10 << 20,
		}),
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}


func (c *Consumer) Run(ctx context.Context) *Stats {
	s := NewStats()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return s // Ctrl+C or SIGTERM
			}
			log.Printf("fetch: %v", err)
			continue
		}

		s.Envelopes++
		s.BytesRead += int64(len(msg.Value))


		if err := fixm.ProcessEnvelope(msg.Value); err != nil {
			s.ParseErrors++
			log.Printf("offset %d: %v", msg.Offset, err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("commit: %v", err)
		}

		if s.Envelopes%1000 == 0 {
			log.Print(s.Progress())
		}
	}
}