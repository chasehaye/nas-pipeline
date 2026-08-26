package kafka

import (
	"context"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type ProducerConfig struct {
	Brokers string
	Topic   string
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(strings.Split(cfg.Brokers, ",")...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) Close() error { return p.writer.Close() }

func (p *Producer) Publish(ctx context.Context, key, data []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{Key: key, Value: data})
}
