package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr           string
	IdentityPath   string
	OperatorPubKey string
	SecretName     string
	SecretNS       string
	MaxAge         time.Duration
	MaxUploadSize  int64
}

func Load() Config {
	return Config{
		Addr:           envOr("LADD_ADMIN_ADDR", ":8092"),
		IdentityPath:   envOr("LADD_ADMIN_IDENTITY_PATH", "/keys/identity.txt"),
		OperatorPubKey: envOr("LADD_OPERATOR_PUBKEY_PATH", "/keys/operator.pub"),
		SecretName:     envOr("LADD_SECRET_NAME", "ladd"),
		SecretNS:       envOr("LADD_SECRET_NAMESPACE", "nas"),
		MaxAge:         envDuration("LADD_MAX_AGE", 9*24*time.Hour),
		MaxUploadSize:  envInt64("LADD_MAX_UPLOAD_BYTES", 4<<20),
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

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
