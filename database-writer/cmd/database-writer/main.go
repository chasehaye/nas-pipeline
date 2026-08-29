package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"
	"github.com/chasehaye/nas-pipeline/platform/observability"

	"github.com/chasehaye/nas-pipeline/database-writer/internal/config"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/kafka"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/pipeline"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/store"
)

func main() {
	// Shared platform: JSON structured logging as the process-wide default.
	observability.InitLogging(os.Getenv("LOG_LEVEL"))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file loaded; using environment and defaults", "err", err)
	}

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// Dead-letter writer for poison (unparseable) messages.
	dlq := kafkax.NewDLQ(cfg.Brokers, cfg.DLQTopic)
	defer dlq.Close()

	// Ops endpoint (/metrics, /healthz, /readyz); readiness pings Kafka and Postgres.
	go observability.Serve(ctx, cfg.OpsAddr, kafkax.ReadinessCheck(cfg.Brokers), st.Ping)

	waitForPostgres(ctx, st)

	if err := st.Migrate(ctx); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.FilteredTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	slog.Info("database-writer started",
		"topic", cfg.FilteredTopic, "group", cfg.Group, "batch", cfg.BatchSize, "flush", cfg.FlushTimeout)

	if err := pipeline.New(consumer, st, dlq, cfg.BatchSize, cfg.FlushTimeout).Run(ctx); err != nil {
		slog.Error("pipeline stopped", "err", err)
		os.Exit(1)
	}
}

func waitForPostgres(ctx context.Context, st *store.Store) {
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := st.Ping(pingCtx)
		cancel()
		if err == nil {
			return
		}
		slog.Warn("postgres not ready; retrying in 2s", "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
