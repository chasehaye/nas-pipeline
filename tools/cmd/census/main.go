package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"
	"github.com/chasehaye/nas-pipeline/platform/observability"

	"github.com/chasehaye/nas-pipeline/tools/internal/accumulator"
	"github.com/chasehaye/nas-pipeline/tools/internal/config"
	"github.com/chasehaye/nas-pipeline/tools/internal/kafka"
	"github.com/chasehaye/nas-pipeline/tools/internal/pipeline"
	"github.com/chasehaye/nas-pipeline/tools/internal/web"
)

func main() {
	observability.InitLogging(os.Getenv("LOG_LEVEL"))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file loaded; using environment and defaults", "err", err)
	}

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	acc := accumulator.New()
	if err := acc.LoadFile(cfg.SnapshotPath); err != nil {
		slog.Warn("could not load snapshot; starting fresh", "path", cfg.SnapshotPath, "err", err)
	} else {
		slog.Info("loaded snapshot", "path", cfg.SnapshotPath)
	}

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.RawTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	go observability.Serve(ctx, cfg.OpsAddr, kafkax.ReadinessCheck(cfg.Brokers))
	go web.Serve(ctx, cfg.HTTPAddr, acc)

	if err := pipeline.New(consumer, acc, cfg.SnapshotPath, cfg.SnapshotEvery).Run(ctx); err != nil {
		slog.Error("census stopped", "err", err)
		os.Exit(1)
	}
}
