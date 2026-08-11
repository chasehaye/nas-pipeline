package router

import (
	"context"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/chasehaye/nas-pipeline/api/internal/handlers"
	"github.com/chasehaye/nas-pipeline/api/internal/router/routes"
	"github.com/chasehaye/nas-pipeline/api/internal/store"
)

func Setup(st *store.Store, corsOrigins []string, reqTimeout time.Duration) *gin.Engine {
	r := gin.New()
	r.Use(
		gin.Recovery(),
		timeoutMiddleware(reqTimeout),
		gzip.Gzip(gzip.DefaultCompression),
		corsMiddleware(corsOrigins),
	)

	health := handlers.NewHealthHandler(st)
	r.GET("/healthz", health.Healthz)


	routes.RedisRouter(r, st)

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
