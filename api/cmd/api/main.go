package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/api/internal/config"
	"github.com/chasehaye/nas-pipeline/api/internal/durable"
	"github.com/chasehaye/nas-pipeline/api/internal/live"
	"github.com/chasehaye/nas-pipeline/api/internal/server"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	liveStore := live.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.KeyPrefix)
	defer liveStore.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := liveStore.Ping(ctx); err != nil {
		log.Fatalf("redis unreachable at %s: %v", cfg.RedisAddr, err)
	}

	// Durable (Postgres) is optional: if it's unreachable the live map still
	// works, the /durable routes are just not registered.
	durableStore := connectDatabase(cfg.DatabaseURL)
	if durableStore != nil {
		defer durableStore.Close()
	}

	log.Printf("Server: listening on %s, reading redis %s (prefix %q)",
		cfg.HTTPAddr, cfg.RedisAddr, cfg.KeyPrefix)

	r := server.Setup(liveStore, durableStore, cfg.CORSOrigins, cfg.RequestTimeout)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}

func connectDatabase(dsn string) *durable.Store {
	if dsn == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d, err := durable.New(ctx, dsn)
	if err != nil {
		log.Printf("durable disabled: postgres connect failed: %v", err)
		return nil
	}
	if err := d.Ping(ctx); err != nil {
		log.Printf("durable disabled: postgres unreachable: %v", err)
		d.Close()
		return nil
	}
	log.Print("durable: postgres reachable, /durable routes enabled")
	return d
}
