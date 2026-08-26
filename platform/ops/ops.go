package ops

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/chasehaye/nas-pipeline/platform/health"
	"github.com/chasehaye/nas-pipeline/platform/metrics"
)

func Serve(ctx context.Context, addr string, ready ...health.Check) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", health.Live)
	mux.Handle("/readyz", health.Ready(ready...))

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	slog.Info("ops endpoint listening", "addr", addr, "paths", "/metrics /healthz /readyz")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("ops server stopped", "err", err)
	}
}
