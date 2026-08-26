package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ReportQueryDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_report_query_duration_seconds",
		Help:    "Cold-path report handler latency by fixed report_key label",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	}, []string{"report_key"})

	ReportErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_report_errors_total",
		Help: "Cold-path report handler errors by fixed report_key and reason labels",
	}, []string{"report_key", "reason"})
)

func PrimeReportMetricLabels(reportKeys, errorReasons []string) {
	for _, key := range reportKeys {
		ReportQueryDurationSeconds.WithLabelValues(key)
		for _, reason := range errorReasons {
			ReportErrorsTotal.WithLabelValues(key, reason)
		}
	}
}
