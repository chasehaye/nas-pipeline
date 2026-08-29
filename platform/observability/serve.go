package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Serve runs the operational endpoint until ctx is cancelled: /metrics for
// Prometheus, /healthz and /readyz for Kubernetes probes. Run it in a goroutine.
func Serve(ctx context.Context, addr string, ready ...Check) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler())
	mux.HandleFunc("/healthz", Live)
	mux.Handle("/readyz", Ready(ready...))

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	slog.Info("observability endpoint listening", "addr", addr, "paths", "/metrics /healthz /readyz")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("observability server stopped", "err", err)
	}
}
