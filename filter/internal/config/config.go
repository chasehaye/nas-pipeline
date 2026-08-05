package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Brokers string

	NormalizedTopic string
	FilteredTopic   string
	Group           string

	LADDDir     string        // active dir: holds the in-effect LADD_Industry_Filter_*.txt; newest date wins
	LADDStaging string        // where a freshly delivered file lands before promotion
	LADDArchive string        // superseded files are moved here and kept for audit
	MaxAge      time.Duration // staleness limit measured from a file's PUBLICATION date
	CheckEvery  time.Duration // how often the reloader promotes from staging and re-scans active
}

func Load() Config {
	// Staging and archive default to siblings of the active dir, so LADD_DIR
	// alone lays out data/ladd/{staging,active,archived}. Either can be pointed
	// elsewhere explicitly.
	active := envOr("LADD_DIR", "./data/ladd/active")
	parent := filepath.Dir(active)

	return Config{
		Brokers:         envOr("KAFKA_BROKERS", "localhost:9092"),
		NormalizedTopic: envOr("KAFKA_TOPIC_NORMALIZED", "fixm.normalized"),
		FilteredTopic:   envOr("KAFKA_TOPIC_FILTERED", "fixm.filtered"),
		Group:           envOr("KAFKA_GROUP", "filter"),

		LADDDir:     active,
		LADDStaging: envOr("LADD_STAGING_DIR", filepath.Join(parent, "staging")),
		LADDArchive: envOr("LADD_ARCHIVE_DIR", filepath.Join(parent, "archived")),
		MaxAge:      envDuration("LADD_MAX_AGE", 9*24*time.Hour),
		CheckEvery:  envDuration("LADD_CHECK_EVERY", time.Hour),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
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
