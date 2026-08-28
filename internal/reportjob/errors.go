package reportjob

import (
	"errors"
	"time"

	"ad-event-processor/internal/metrics"
)

var ErrForbidden = errors.New("forbidden")

const (
	defaultReportLookback = 7 * 24 * time.Hour
	maxStatsRange         = 90 * 24 * time.Hour
)

func observeReportQuery(reportKey string, start time.Time, err error) {
	metrics.ReportQueryDurationSeconds.WithLabelValues(reportKey).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ReportErrorsTotal.WithLabelValues(reportKey, "internal").Inc()
	}
}
