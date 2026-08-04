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
			Addr: kafka.TCP(
				strings.Split(cfg.Brokers, ",")...,
			),

			Topic: cfg.Topic,

			Balancer: &kafka.LeastBytes{},

			// Default BatchTimeout is 1s, so a single synchronous write
			// stalls ~1s waiting for a batch that never fills. Flush fast.
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}


func (p *Producer) Close() error {
	return p.writer.Close()
}


func (p *Producer) Publish(
	ctx context.Context,
	data []byte,
) error {

	return p.writer.WriteMessages(
		ctx,
		kafka.Message{
			Value: data,
		},
	)
}