package controlplane

import (
	"testing"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseFraudEvalReportFromPGRow(t *testing.T) {
	generatedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	report := parseFraudEvalReportFromPGRow(
		"ok",
		generatedAt,
		0.91,
		0.42,
		[]byte(`{"drift_detected": true, "max_drift": 0.35}`),
		"proxy",
	)
	require.True(t, report.Available)
	require.Equal(t, "ok", report.Status)
	require.InDelta(t, 0.91, report.Precision, 0.001)
	require.True(t, report.DriftDetected)
}

func TestPublishFraudMLEvalMetrics(t *testing.T) {
	publishFraudMLEvalMetrics(fraudEvalReport{
		Available:     true,
		Status:        "ok",
		Precision:     0.88,
		Recall:        0.41,
		DriftDetected: true,
		GeneratedTime: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
	})
	require.InDelta(t, 0.88, testutil.ToFloat64(metrics.FraudMLShadowPrecision), 0.001)
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.FraudMLDriftDetected))
}
