package controlplane

import (
	"net/http"
	"time"

	"ad-event-processor/internal/metrics"
)

func init() {
	metrics.PrimeReportMetricLabels(LiveReportMetricKeys(), reportErrorReasons())
}

type reportStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *reportStatusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (reports *ReportsHTTPHandlers) wrapReport(reportKey string, next http.HandlerFunc) http.HandlerFunc {
	if next == nil {
		return nil
	}
	return observeReportHandler(reportKey, next)
}

func observeReportHandler(reportKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &reportStatusWriter{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		observeReportQuery(reportKey, start, nil)
		if rec.status >= http.StatusBadRequest {
			metrics.ReportErrorsTotal.WithLabelValues(reportKey, reportErrorReason(rec.status)).Inc()
		}
	}
}

func observeReportQuery(reportKey string, start time.Time, err error) {
	metrics.ReportQueryDurationSeconds.WithLabelValues(reportKey).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ReportErrorsTotal.WithLabelValues(reportKey, "internal").Inc()
	}
}

func reportErrorReason(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusServiceUnavailable:
		return "ch_unavailable"
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return "query_timeout"
	default:
		return "internal"
	}
}
