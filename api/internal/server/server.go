package server

import (
	"context"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/chasehaye/nas-pipeline/platform/observability"

	"github.com/chasehaye/nas-pipeline/api/internal/durable"
	"github.com/chasehaye/nas-pipeline/api/internal/live"
	"github.com/chasehaye/nas-pipeline/api/internal/metrics"
)

func Setup(
	liveStore *live.Store,
	durableStore *durable.Store,
	corsOrigins []string,
	reqTimeout time.Duration,
) *gin.Engine {
	r := gin.New()
	r.Use(
		gin.Recovery(),
		metrics.Middleware(), // record RED metrics for every request
		timeoutMiddleware(reqTimeout),
		// Exclude /metrics: promhttp gzips itself; compressing again double-gzips
		// the scrape and Prometheus can't parse it.
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics"})),
		corsMiddleware(corsOrigins),
	)

	// Ops surface on the same server; readiness gates on the stores.
	r.GET("/metrics", gin.WrapH(observability.MetricsHandler()))
	r.GET("/healthz", gin.WrapF(observability.Live))

	checks := []observability.Check{liveStore.Ping}
	if durableStore != nil {
		checks = append(checks, durableStore.Ping)
	}
	r.GET("/readyz", gin.WrapF(observability.Ready(checks...)))

	live.Routes(r, liveStore)

	// Durable (Postgres) routes are registered only when it's reachable.
	if durableStore != nil {
		durable.Routes(r, durableStore)
	}

	return r
}

func timeoutMiddleware(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}
	if len(origins) == 0 || contains(origins, "*") {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = origins
	}
	return cors.New(cfg)
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
