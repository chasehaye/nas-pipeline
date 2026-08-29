// Package observability provides a service's operational surface: structured
// logging plus the /metrics, /healthz, and /readyz endpoints.
package observability

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogging installs a JSON slog handler as the process default at the given
// level ("debug"|"info"|"warn"|"error"; anything else = info).
func InitLogging(level string) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
