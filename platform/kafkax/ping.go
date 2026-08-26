package kafkax

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

func Ping(ctx context.Context, brokers string) error {
	addr := strings.TrimSpace(strings.Split(brokers, ",")[0])
	conn, err := kafka.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}
