package config

import (
	"os"
	"time"
)

type Config struct {
	Brokers  string
	RawTopic string
	Group    string // its own consumer group, independent of the pipeline's offsets

	SnapshotPath  string        // where the accumulated shape is persisted as JSON
	SnapshotEvery time.Duration // how often to write the snapshot (and commit offsets)

	HTTPAddr string // the shape web UI (/ and /shape)
	OpsAddr  string // /metrics, /healthz, /readyz
}

func Load() Config {
	return Config{
		Brokers:  envOr("KAFKA_BROKERS", "localhost:9092"),
		RawTopic: envOr("KAFKA_TOPIC_RAW", "fixm.raw"),
		Group:    envOr("KAFKA_GROUP", "census"),

		SnapshotPath:  envOr("CENSUS_SNAPSHOT", "census.json"),
		SnapshotEvery: envDuration("CENSUS_SNAPSHOT_EVERY", 30*time.Second),

		HTTPAddr: envOr("CENSUS_HTTP_ADDR", ":8088"),
		OpsAddr:  envOr("OPS_ADDR", ":2116"), // next in the pipeline's local sequence (2112-2115); k8s overrides to :2112
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
