package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"
	"github.com/chasehaye/nas-pipeline/platform/log"
	"github.com/chasehaye/nas-pipeline/platform/ops"

	"github.com/chasehaye/nas-pipeline/processor/internal/config"
	"github.com/chasehaye/nas-pipeline/processor/internal/kafka"
	"github.com/chasehaye/nas-pipeline/processor/internal/pipeline"
)

func main() {
	// Shared platform: JSON structured logging as the process-wide default.
	log.Init(os.Getenv("LOG_LEVEL"))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file loaded; using environment and defaults", "err", err)
	}

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ops endpoint (/metrics, /healthz, /readyz); readiness pings Kafka.
	go ops.Serve(ctx, cfg.OpsAddr, func(c context.Context) error {
		return kafkax.Ping(c, cfg.Brokers)
	})

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.RawTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	producer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.NormalizedTopic,
	})
	defer producer.Close()

	// Dead-letter writer for poison (unparseable) envelopes.
	dlq := kafkax.NewDLQ(cfg.Brokers, cfg.DLQTopic)
	defer dlq.Close()

	if err := pipeline.New(consumer, producer, dlq, cfg.Workers).Run(ctx); err != nil {
		slog.Error("pipeline stopped", "err", err)
		os.Exit(1)
	}
}
