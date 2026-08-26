package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus collectors for this service, served via platform/metrics.
var (
	EnvelopesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "normalizer_envelopes_processed_total",
		Help: "Raw FIXM envelopes consumed and processed.",
	})
	FlightsPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "normalizer_flights_published_total",
		Help: "Per-flight messages published to fixm.normalized.",
	})
	ParseErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "normalizer_parse_errors_total",
		Help: "Envelopes dropped because they could not be parsed (poison).",
	})
	PublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "normalizer_publish_errors_total",
		Help: "Failed publishes to fixm.normalized (transient; gates the commit).",
	})
	DLQPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "normalizer_dlq_published_total",
		Help: "Poison messages dead-lettered to fixm.normalized.dlq.",
	})
	ProcessDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "normalizer_process_duration_seconds",
		Help:    "Wall time to parse and publish one envelope.",
		Buckets: prometheus.DefBuckets,
	})
)
