package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/redis-service/internal/config"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/kafka"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/pipeline"
	"github.com/chasehaye/nas-pipeline/redis-service/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded (%v); using environment and defaults", err)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	waitForRedis(ctx, st)

	log.Printf("redis-service: %s -> redis %s (group %q, ttl %s)",
		cfg.FilteredTopic, cfg.RedisAddr, cfg.Group, cfg.FlightTTL)

	if err := pipeline.New(consumer, st).Run(ctx); err != nil {
		log.Fatal(err)
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
		log.Printf("redis not ready (%v); retrying in 2s", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
