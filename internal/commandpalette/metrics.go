package commandpalette

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SearchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "command_palette_search_total",
		Help: "Command palette search requests by outcome",
	}, []string{"outcome"})

	SearchErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "command_palette_search_errors_total",
		Help: "Command palette search errors by reason",
	}, []string{"reason"})

	SearchDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "command_palette_search_duration_seconds",
		Help:    "Command palette search handler latency",
		Buckets: prometheus.DefBuckets,
	})

	OpenTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "command_palette_open_total",
		Help: "Command palette open events reported by admin UI",
	}, []string{"source"})
)

func IncSearchError(reason string) {
	if reason == "" {
		reason = "unknown"
	}
	SearchErrorsTotal.WithLabelValues(reason).Inc()
}
