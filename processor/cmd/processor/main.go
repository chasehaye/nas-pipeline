package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/processor/internal/config"
	"github.com/chasehaye/nas-pipeline/processor/internal/kafka"
	"github.com/chasehaye/nas-pipeline/processor/internal/pipeline"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded (%v); using environment and defaults", err)
	}

	cfg := config.Load()

	if err := kafka.EnsureTopic(cfg.Brokers, cfg.NormalizedTopic, 1, 1); err != nil {
		log.Fatal(err)
	}

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

	if err := pipeline.New(consumer, producer).Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
