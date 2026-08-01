package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chasehaye/nas-pipeline/processor/internal/consumer"
)

func main() {
	c := consumer.New(consumer.Config{
		Brokers: envOr("KAFKA_BROKERS", "localhost:9092"),
		Topic:   envOr("KAFKA_TOPIC_RAW", "fixm.raw"),
		Group:   envOr("KAFKA_GROUP", "processor"),
	})
	defer c.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	s := c.Run(ctx)
	log.Print(s.Summary())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}