package postback

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	dispatchTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_postback_dispatch_total",
		Help: "Outbound CAPI/webhook dispatch attempts by provider and status",
	}, []string{"provider", "status"})

	dispatchDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_postback_dispatch_duration_seconds",
		Help:    "Wall time of outbound CAPI/webhook Send calls",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"provider"})

	dispatchDuplicatesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ad_postback_dispatch_duplicates_total",
		Help: "SEND_POSTBACK outbox events skipped as duplicates via postback_dispatches hash",
	})

	dispatchDLQTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_postback_dlq_total",
		Help: "Postback events moved to DLQ after retry exhaustion",
	}, []string{"provider"})
)

func RegisterMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(
			dispatchTotal,
			dispatchDurationSeconds,
			dispatchDuplicatesTotal,
			dispatchDLQTotal,
		)
	})
}

func recordDispatch(provider, status string, durationSeconds float64) {
	p := normalizeProviderLabel(provider)
	dispatchTotal.WithLabelValues(p, status).Inc()
	if durationSeconds >= 0 {
		dispatchDurationSeconds.WithLabelValues(p).Observe(durationSeconds)
	}
}

func recordDuplicate() {
	dispatchDuplicatesTotal.Inc()
}

func recordDLQ(provider string) {
	dispatchDLQTotal.WithLabelValues(normalizeProviderLabel(provider)).Inc()
}

func normalizeProviderLabel(provider string) string {
	if provider == "" {
		return "unknown"
	}
	return provider
}
