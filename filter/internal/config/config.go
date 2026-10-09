package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Brokers string

	NormalizedTopic string
	FilteredTopic   string
	DLQTopic        string // dead-letter topic for poison (unparseable) messages
	Group           string

	LADDDir     string        // active dir: holds the in-effect LADD_Industry_Filter_*.txt; newest date wins
	LADDStaging string        // where a freshly delivered file lands before promotion
	LADDArchive string        // superseded files are moved here and kept for audit
	MaxAge      time.Duration // staleness limit measured from a file's PUBLICATION date
	CheckEvery  time.Duration // how often the reloader promotes from staging and re-scans active

	Workers int    // number of concurrent processing workers
	OpsAddr string // host:port for the ops endpoint (/metrics, /healthz, /readyz)

	// IsDev mirrors the bridge's ISDEV switch. Optional, defaults to false. When
	// true the filter runs in dev mode: it skips LADD loading and the
	// fail-closed staleness gate and forwards everything, because the dev stream
	// is already LADD-clean. No CUI LADD list is required locally.
	IsDev bool
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
		DLQTopic:        envOr("KAFKA_TOPIC_FILTERED_DLQ", "fixm.filtered.dlq"),
		Group:           envOr("KAFKA_GROUP", "filter"),

		LADDDir:     active,
		LADDStaging: envOr("LADD_STAGING_DIR", filepath.Join(parent, "staging")),
		LADDArchive: envOr("LADD_ARCHIVE_DIR", filepath.Join(parent, "archived")),
		MaxAge:      envDuration("LADD_MAX_AGE", 9*24*time.Hour),
		CheckEvery:  envDuration("LADD_CHECK_EVERY", time.Hour),

		Workers: envInt("FILTER_WORKERS", 1000),
		OpsAddr: envOr("OPS_ADDR", ":2113"),

		IsDev: envBool("ISDEV", false),
	}
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
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
