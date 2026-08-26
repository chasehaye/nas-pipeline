package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Brokers       string
	FilteredTopic string
	DLQTopic      string // dead-letter topic for poison (unparseable) messages
	Group         string
	DatabaseURL   string

	BatchSize    int
	FlushTimeout time.Duration

	OpsAddr string // host:port for the ops endpoint (/metrics, /healthz, /readyz)
}

func Load() Config {
	return Config{
		Brokers:       envOr("KAFKA_BROKERS", "localhost:9092"),
		FilteredTopic: envOr("KAFKA_TOPIC_FILTERED", "fixm.filtered"),
		DLQTopic:      envOr("KAFKA_TOPIC_DB_DLQ", "database-writer.dlq"),
		Group:         envOr("KAFKA_GROUP", "database-writer"),
		DatabaseURL:   envOr("DATABASE_URL", "postgres://naspipeline:changeme@localhost:5433/naspipeline?sslmode=disable"),
		BatchSize:     envInt("DB_WRITER_BATCH_SIZE", 1000),
		FlushTimeout:  envDuration("DB_WRITER_FLUSH_TIMEOUT", time.Second),
		OpsAddr:       envOr("OPS_ADDR", ":2115"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
