package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/api/internal/config"
	"github.com/chasehaye/nas-pipeline/api/internal/router"
	"github.com/chasehaye/nas-pipeline/api/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	st := store.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.KeyPrefix)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := st.Ping(ctx); err != nil {
		log.Fatalf("redis unreachable at %s: %v", cfg.RedisAddr, err)
	}

	log.Printf("Server: listening on %s, reading redis %s (prefix %q)",
		cfg.HTTPAddr, cfg.RedisAddr, cfg.KeyPrefix)

	r := router.Setup(st, cfg.CORSOrigins, cfg.RequestTimeout)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
