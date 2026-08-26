package config

import (
	"os"
	"runtime"
	"strconv"
)

type Config struct {
	Brokers string

	RawTopic        string
	NormalizedTopic string
	DLQTopic        string // dead-letter topic for poison (unparseable) envelopes
	Group           string

	Workers int // number of concurrent processing workers (CPU-bound; ~NumCPU)

	OpsAddr string // host:port for the ops endpoint (/metrics, /healthz, /readyz)
}

func Load() Config {
	return Config{
		Brokers: envOr(
			"KAFKA_BROKERS",
			"localhost:9092",
		),
		RawTopic: envOr(
			"KAFKA_TOPIC_RAW",
			"fixm.raw",
		),
		NormalizedTopic: envOr(
			"KAFKA_TOPIC_NORMALIZED",
			"fixm.normalized",
		),
		DLQTopic: envOr(
			"KAFKA_TOPIC_NORMALIZED_DLQ",
			"fixm.normalized.dlq",
		),
		Group: envOr(
			"KAFKA_GROUP",
			"normalizer",
		),
		Workers: envInt("NORMALIZER_WORKERS", runtime.NumCPU()),
		OpsAddr: envOr("OPS_ADDR", ":2112"),
	}
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}

	return fallback
}
