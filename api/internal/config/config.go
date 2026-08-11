package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTPAddr string

	RequestTimeout time.Duration // per-request deadline, applied as middleware

	RedisAddr     string
	RedisPassword string
	RedisDB       int
	KeyPrefix     string // must match the cache-writer's REDIS_KEY_PREFIX

	CORSOrigins []string // allowed browser origins; ["*"] = any
}

func Load() Config {
	return Config{
		Env:      envOr("GO_ENV", "dev"),
		HTTPAddr: envOr("HTTP_ADDR", ":8090"),

		RequestTimeout: envDuration("REQUEST_TIMEOUT", 5*time.Second),

		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword: envOr("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		KeyPrefix:     envOr("REDIS_KEY_PREFIX", "flight:"),

		CORSOrigins: splitCSV(envOr("CORS_ORIGINS", "*")),
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

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
