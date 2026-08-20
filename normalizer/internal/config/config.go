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
	Group           string

	Workers int // number of concurrent processing workers (CPU-bound; ~NumCPU)
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
		Group: envOr(
			"KAFKA_GROUP",
			"processor",
		),
		Workers: envInt("NORMALIZER_WORKERS", runtime.NumCPU()),
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