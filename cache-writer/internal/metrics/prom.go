package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus collectors for this service, served via platform/metrics.
var (
	Messages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_messages_total",
		Help: "Messages consumed from fixm.filtered.",
	})
	Stored = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_stored_total",
		Help: "Active flights upserted into Redis.",
	})
	Removed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_removed_total",
		Help: "Non-active flights deleted from Redis.",
	})
	NoPosition = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_no_position_total",
		Help: "Active flights skipped because they carried no position.",
	})
	ParseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_parse_errors_total",
		Help: "Messages dropped because they could not be parsed (poison).",
	})
	WriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_write_errors_total",
		Help: "Redis write failures after retries (transient; not committed).",
	})
	DLQPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_writer_dlq_published_total",
		Help: "Poison messages dead-lettered.",
	})
	ProcessDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "cache_writer_process_duration_seconds",
		Help:    "Wall time to parse and apply one message to Redis.",
		Buckets: prometheus.DefBuckets,
	})
)
