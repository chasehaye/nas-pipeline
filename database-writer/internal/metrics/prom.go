package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus collectors for this service, served via platform/metrics.
var (
	Messages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_messages_total",
		Help: "Messages consumed from fixm.filtered.",
	})
	Recorded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_recorded_total",
		Help: "Flights written to Postgres.",
	})
	Positions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_positions_total",
		Help: "Flights that carried a position (a positions-table row).",
	})
	Skipped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_skipped_total",
		Help: "Messages skipped (e.g. missing GUFI).",
	})
	ParseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_parse_errors_total",
		Help: "Messages dropped because they could not be parsed (poison).",
	})
	DLQPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_dlq_published_total",
		Help: "Poison messages dead-lettered.",
	})
	BatchWriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "database_writer_batch_write_errors_total",
		Help: "Batch write attempts that failed (transient; retried until durable).",
	})
	BatchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "database_writer_batch_duration_seconds",
		Help:    "Wall time for one successful batch write transaction.",
		Buckets: prometheus.DefBuckets,
	})
)
