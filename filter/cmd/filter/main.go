package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/kafkax"
	"github.com/chasehaye/nas-pipeline/platform/observability"

	"github.com/chasehaye/nas-pipeline/filter/internal/config"
	"github.com/chasehaye/nas-pipeline/filter/internal/kafka"
	"github.com/chasehaye/nas-pipeline/filter/internal/ladd"
	"github.com/chasehaye/nas-pipeline/filter/internal/pipeline"
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

	// Ops endpoint (/metrics, /healthz, /readyz); readiness pings Kafka.
	go observability.Serve(ctx, cfg.OpsAddr, kafkax.ReadinessCheck(cfg.Brokers))

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.NormalizedTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	producer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.FilteredTopic,
	})
	defer producer.Close()

	// Dead-letter writer for poison (unparseable) messages.
	dlq := kafkax.NewDLQ(cfg.Brokers, cfg.DLQTopic)
	defer dlq.Close()

	store := ladd.NewStore(cfg.MaxAge)

	dirs := ladd.Dirs{
		Staging:  cfg.LADDStaging,
		Active:   cfg.LADDDir,
		Archived: cfg.LADDArchive,
	}

	if promoted, err := ladd.Promote(dirs); err != nil {
		slog.Warn("LADD promote on startup failed", "err", err)
	} else if promoted != "" {
		slog.Info("LADD promoted from staging", "file", promoted)
	}

	if set, effective, err := ladd.LoadLatest(dirs.Active); err != nil {
		slog.Warn("LADD list not loaded", "dir", dirs.Active, "err", err)
	} else {
		store.Swap(set, effective)
		slog.Info("LADD list loaded", "entries", set.Len(), "effective", effective.Format("2006-01-02"))
	}

	go func() {
		t := time.NewTicker(cfg.CheckEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				swapped, err := store.Reload(dirs)
				if err != nil {
					slog.Warn("ladd reload issue (keeping current list)", "err", err)
				}
				if swapped {
					slog.Info("ladd list reloaded (newer file promoted and picked up)")
				}
			}
		}
	}()

	if err := pipeline.New(consumer, producer, dlq, store, cfg.Workers).Run(ctx); err != nil {
		slog.Error("pipeline stopped", "err", err)
		os.Exit(1)
	}
}
