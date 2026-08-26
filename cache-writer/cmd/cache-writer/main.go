package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"
	"github.com/chasehaye/nas-pipeline/platform/log"
	"github.com/chasehaye/nas-pipeline/platform/ops"

	"github.com/chasehaye/nas-pipeline/redis-service/internal/config"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/kafka"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/pipeline"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/store"
)

func main() {
	// Shared platform: JSON structured logging as the process-wide default.
	log.Init(os.Getenv("LOG_LEVEL"))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file loaded; using environment and defaults", "err", err)
	}

	cfg := config.Load()

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.FilteredTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	st := store.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.KeyPrefix, cfg.FlightTTL)
	defer st.Close()

	// Dead-letter writer for poison (unparseable) messages.
	dlq := kafkax.NewDLQ(cfg.Brokers, cfg.DLQTopic)
	defer dlq.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ops endpoint (/metrics, /healthz, /readyz); readiness pings Kafka and Redis.
	go ops.Serve(ctx, cfg.OpsAddr,
		func(c context.Context) error { return kafkax.Ping(c, cfg.Brokers) },
		func(c context.Context) error { return st.Ping(c) },
	)

	waitForRedis(ctx, st)

	slog.Info("cache-writer started",
		"topic", cfg.FilteredTopic, "redis", cfg.RedisAddr, "group", cfg.Group, "ttl", cfg.FlightTTL)

	if err := pipeline.New(consumer, st, dlq).Run(ctx); err != nil {
		slog.Error("pipeline stopped", "err", err)
		os.Exit(1)
	}
}

func waitForRedis(ctx context.Context, st *store.Store) {
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := st.Ping(pingCtx)
		cancel()
		if err == nil {
			return
		}
		slog.Warn("redis not ready; retrying in 2s", "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
