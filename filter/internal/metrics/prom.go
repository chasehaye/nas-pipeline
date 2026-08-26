package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus collectors for this service, served via platform/metrics.
var (
	FlightsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_flights_total",
		Help: "Flights consumed from fixm.normalized.",
	})
	Forwarded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_forwarded_total",
		Help: "Flights passed the LADD filter and published to fixm.filtered.",
	})
	Blocked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_blocked_total",
		Help: "Flights suppressed by the LADD block list (compliance).",
	})
	ParseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_parse_errors_total",
		Help: "Messages dropped because they could not be parsed (poison).",
	})
	PublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_publish_errors_total",
		Help: "Failed publishes to fixm.filtered (transient; gates the commit).",
	})
	DLQPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_dlq_published_total",
		Help: "Poison messages dead-lettered to fixm.filtered.dlq.",
	})
	ProcessDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "filter_process_duration_seconds",
		Help:    "Wall time to parse, check the block list, and publish one flight.",
		Buckets: prometheus.DefBuckets,
	})
)
