package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RED metrics (Rate, Errors, Duration) for the HTTP API.
var (
	Requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_http_requests_total",
		Help: "HTTP requests handled, by method, route and status.",
	}, []string{"method", "route", "status"})

	Duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_http_request_duration_seconds",
		Help:    "HTTP request duration by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// Middleware records count and duration per request, labelled by route template
// (c.FullPath) not raw path, to keep label cardinality bounded.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		Requests.WithLabelValues(c.Request.Method, route, status).Inc()
		Duration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}
