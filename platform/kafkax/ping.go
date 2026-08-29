package kafkax

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

// Ping dials a broker to verify Kafka is reachable.
func Ping(ctx context.Context, brokers string) error {
	addr := strings.TrimSpace(brokers)
	conn, err := kafka.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

// ReadinessCheck adapts Ping into a readiness check (func(ctx) error)
// for observability.Serve.
func ReadinessCheck(brokers string) func(context.Context) error {
	return func(ctx context.Context) error {
		return Ping(ctx, brokers)
	}
}
