package kafkax

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Envelope struct {
	OriginalTopic string    `json:"original_topic"`
	Partition     int       `json:"partition"`
	Offset        int64     `json:"offset"`
	FailedAt      time.Time `json:"failed_at"`
	Stage         string    `json:"stage"`
	ErrorClass    string    `json:"error_class"`
	Error         string    `json:"error"`
	Payload       []byte    `json:"payload"`
}

type DLQ struct {
	w *kafka.Writer
}

func NewDLQ(brokers, topic string) *DLQ {
	return &DLQ{w: &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}}
}

func (d *DLQ) Publish(ctx context.Context, orig kafka.Message, stage, class, cause string) error {
	body, err := json.Marshal(Envelope{
		OriginalTopic: orig.Topic,
		Partition:     orig.Partition,
		Offset:        orig.Offset,
		FailedAt:      time.Now().UTC(),
		Stage:         stage,
		ErrorClass:    class,
		Error:         cause,
		Payload:       orig.Value,
	})
	if err != nil {
		return err
	}
	return d.w.WriteMessages(ctx, kafka.Message{Value: body})
}

func (d *DLQ) Close() error { return d.w.Close() }
