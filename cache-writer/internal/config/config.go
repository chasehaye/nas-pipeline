package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Brokers       string
	FilteredTopic string
	Group         string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// FlightTTL is how long a flight stays "active" without a fresh position.
	// Every position update re-arms the key's expiry; a flight that lands or
	// stops reporting simply ages out, which is what makes the key set == the
	// set of flights currently in the air.
	FlightTTL time.Duration
	KeyPrefix string // redis key prefix, e.g. "flight:"
}

func Load() Config {
	return Config{
		// Consume the LADD-filtered stream: blocked aircraft are already gone,
		// so the active-flights view is compliant by construction.
		Brokers:       envOr("KAFKA_BROKERS", "localhost:9092"),
		FilteredTopic: envOr("KAFKA_TOPIC_FILTERED", "fixm.filtered"),
		Group:         envOr("KAFKA_GROUP", "redis-writer"),

		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword: envOr("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		FlightTTL: envDuration("FLIGHT_TTL", 10*time.Minute),
		KeyPrefix: envOr("REDIS_KEY_PREFIX", "flight:"),
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
