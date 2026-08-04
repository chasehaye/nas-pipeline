package kafka

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)


type ConsumerConfig struct {
	Brokers string
	Topic   string
	Group   string
}


type Consumer struct {
	reader *kafka.Reader
}


func NewConsumer(cfg ConsumerConfig) *Consumer {

	return &Consumer{

		reader: kafka.NewReader(
			kafka.ReaderConfig{
				Brokers: strings.Split(cfg.Brokers, ","),

				Topic: cfg.Topic,

				GroupID: cfg.Group,

				MinBytes: 1,

				MaxBytes: 10 << 20,
			},
		),
	}
}



func (c *Consumer) Close() error {
	return c.reader.Close()
}



func (c *Consumer) Fetch(
	ctx context.Context,
) (kafka.Message, error) {

	return c.reader.FetchMessage(ctx)
}



func (c *Consumer) Commit(
	ctx context.Context,
	msg kafka.Message,
) error {

	return c.reader.CommitMessages(
		ctx,
		msg,
	)
}