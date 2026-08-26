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

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	FlightTTL time.Duration
	KeyPrefix string // redis key prefix, e.g. "flight:"

	OpsAddr string // host:port for the ops endpoint (/metrics, /healthz, /readyz)
}

func Load() Config {
	return Config{
		Brokers:       envOr("KAFKA_BROKERS", "localhost:9092"),
		FilteredTopic: envOr("KAFKA_TOPIC_FILTERED", "fixm.filtered"),
		DLQTopic:      envOr("KAFKA_TOPIC_CACHE_DLQ", "cache-writer.dlq"),
		Group:         envOr("KAFKA_GROUP", "redis-writer"),

		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword: envOr("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		FlightTTL: envDuration("FLIGHT_TTL", 3*time.Minute),
		KeyPrefix: envOr("REDIS_KEY_PREFIX", "flight:"),

		OpsAddr: envOr("OPS_ADDR", ":2114"),
	}
}

func envOr(key, fallback string) string {
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

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
