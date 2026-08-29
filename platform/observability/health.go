package observability

import (
	"context"
	"net/http"
	"time"
)

// Check reports whether a dependency is usable; nil means healthy.
type Check func(ctx context.Context) error

// Live is the liveness handler: 200 as long as the server can respond.
func Live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Ready runs every check and reports 200 only if all pass, else 503.
func Ready(checks ...Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		for _, c := range checks {
			if err := c(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}
}
