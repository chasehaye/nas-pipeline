// Package web serves the live shape: /shape returns the accumulator snapshot as
// JSON, / serves a small page that renders it as a table.
package web

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/chasehaye/nas-pipeline/tools/internal/accumulator"
)

//go:embed index.html
var indexHTML []byte

func Serve(ctx context.Context, addr string, acc *accumulator.Accumulator) {
	mux := http.NewServeMux()

	mux.HandleFunc("/shape", func(w http.ResponseWriter, _ *http.Request) {
		data, err := acc.MarshalSnapshot()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	slog.Info("census web listening", "addr", addr, "paths", "/ /shape")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("census web server stopped", "err", err)
	}
}
