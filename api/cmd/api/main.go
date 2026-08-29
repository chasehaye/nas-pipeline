package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/observability"

	"github.com/chasehaye/nas-pipeline/api/internal/config"
	"github.com/chasehaye/nas-pipeline/api/internal/durable"
	"github.com/chasehaye/nas-pipeline/api/internal/live"
	"github.com/chasehaye/nas-pipeline/api/internal/server"
)

func main() {
	// Shared platform: JSON structured logging as the process-wide default.
	observability.InitLogging(os.Getenv("LOG_LEVEL"))

	_ = godotenv.Load()

	cfg := config.Load()

	liveStore := live.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.KeyPrefix)
	defer liveStore.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := liveStore.Ping(ctx); err != nil {
		slog.Error("redis unreachable", "addr", cfg.RedisAddr, "err", err)
		os.Exit(1)
	}

	// Durable (Postgres) is optional: if it's unreachable the live map still
	// works, the /durable routes are just not registered.
	durableStore := connectDatabase(cfg.DatabaseURL)
	if durableStore != nil {
		defer durableStore.Close()
	}

	slog.Info("api starting",
		"addr", cfg.HTTPAddr, "redis", cfg.RedisAddr, "prefix", cfg.KeyPrefix)

	r := server.Setup(liveStore, durableStore, cfg.CORSOrigins, cfg.RequestTimeout)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
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
		slog.Warn("durable disabled: postgres connect failed", "err", err)
		return nil
	}
	if err := d.Ping(ctx); err != nil {
		slog.Warn("durable disabled: postgres unreachable", "err", err)
		d.Close()
		return nil
	}
	slog.Info("durable: postgres reachable, /durable routes enabled")
	return d
}
