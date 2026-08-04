package config

import "os"

type Config struct {
	Brokers string

	RawTopic        string
	NormalizedTopic string
	Group           string
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
	}
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}