package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/database-writer/internal/config"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/kafka"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/pipeline"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded (%v); using environment and defaults", err)
	}

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer st.Close()

	waitForPostgres(ctx, st)

	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.FilteredTopic,
		Group:   cfg.Group,
	})
	defer consumer.Close()

	log.Printf("database-writer: %s -> postgres (group %q, batch %d / %s)",
		cfg.FilteredTopic, cfg.Group, cfg.BatchSize, cfg.FlushTimeout)

	if err := pipeline.New(consumer, st, cfg.BatchSize, cfg.FlushTimeout).Run(ctx); err != nil {
		log.Fatal(err)
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
		log.Printf("postgres not ready (%v); retrying in 2s", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
