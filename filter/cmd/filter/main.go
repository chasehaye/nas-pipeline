package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/filter/internal/config"
	"github.com/chasehaye/nas-pipeline/filter/internal/kafka"
	"github.com/chasehaye/nas-pipeline/filter/internal/ladd"
	"github.com/chasehaye/nas-pipeline/filter/internal/pipeline"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded (%v); using environment and defaults", err)
	}

	cfg := config.Load()

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

	store := ladd.NewStore(cfg.MaxAge)

	dirs := ladd.Dirs{
		Staging:  cfg.LADDStaging,
		Active:   cfg.LADDDir,
		Archived: cfg.LADDArchive,
	}


	if promoted, err := ladd.Promote(dirs); err != nil {
		log.Printf("LADD promote on startup failed (%v)", err)
	} else if promoted != "" {
		log.Printf("LADD promoted from staging: %s", promoted)
	}

	if set, effective, err := ladd.LoadLatest(dirs.Active); err != nil {
		log.Printf("LADD list not loaded from %s (%v)", dirs.Active, err)
	} else {
		store.Swap(set, effective)
		log.Printf("LADD list loaded: %d entries, effective %s", set.Len(), effective.Format("2006-01-02"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
					log.Printf("ladd reload issue (keeping current list): %v", err)
				}
				if swapped {
					log.Print("ladd list reloaded (newer file promoted and picked up)")
				}
			}
		}
	}()

	if err := pipeline.New(consumer, producer, store).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
