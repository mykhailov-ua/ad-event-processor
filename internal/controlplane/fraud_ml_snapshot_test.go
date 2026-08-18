package controlplane

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeMLEvalStatus(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	t.Run("healthy", func(t *testing.T) {
		status := computeMLEvalStatus(fraudEvalReport{
			Available:     true,
			Status:        "ok",
			GeneratedAt:   now.Format(time.RFC3339),
			GeneratedTime: now,
		}, false)
		require.Equal(t, "healthy", status)
	})

	t.Run("eval_stale", func(t *testing.T) {
		status := computeMLEvalStatus(fraudEvalReport{Available: true, Status: "ok"}, true)
		require.Equal(t, "eval_stale", status)
	})

	t.Run("drift_detected", func(t *testing.T) {
		status := computeMLEvalStatus(fraudEvalReport{
			Available:     true,
			Status:        "ok",
			DriftDetected: true,
		}, false)
		require.Equal(t, "drift_detected", status)
	})

	t.Run("eval_unavailable_on_error_status", func(t *testing.T) {
		status := computeMLEvalStatus(fraudEvalReport{Available: true, Status: "error"}, false)
		require.Equal(t, "eval_unavailable", status)
	})

	t.Run("eval_unavailable_on_empty", func(t *testing.T) {
		status := computeMLEvalStatus(fraudEvalReport{Status: "empty"}, false)
		require.Equal(t, "eval_unavailable", status)
	})
}

func TestLoadFraudEvalReport_parsesJSON(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/shadow_eval_report.json"
	content := `{
		"status": "ok",
		"generated_at": "2026-02-01T10:00:00Z",
		"label_method": "proxy",
		"precision": 0.91,
		"recall": 0.42,
		"drift": {"drift_detected": true, "max_drift": 0.35, "status": "ok"}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("FRAUD_EVAL_REPORT_PATH", path)

	fraudEvalReportCache.mu.Lock()
	fraudEvalReportCache.loadedAt = time.Time{}
	fraudEvalReportCache.path = ""
	fraudEvalReportCache.mu.Unlock()

	report := loadFraudEvalReportFromFile(time.Now().UTC())
	require.True(t, report.Available)
	require.Equal(t, "proxy", report.LabelMethod)
	require.InDelta(t, 0.91, report.Precision, 0.001)
	require.True(t, report.DriftDetected)
	require.Contains(t, report.DriftSummary, "30%")
}

func TestFraudMLSnapshotDTOFields(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/shadow_eval_report.json"
	require.NoError(t, os.WriteFile(path, []byte(`{
		"status": "ok",
		"generated_at": "2026-02-01T10:00:00Z",
		"label_method": "proxy",
		"precision": 0.8,
		"recall": 0.5
	}`), 0o644))
	t.Setenv("FRAUD_EVAL_REPORT_PATH", path)

	fraudEvalReportCache.mu.Lock()
	fraudEvalReportCache.loadedAt = time.Time{}
	fraudEvalReportCache.path = ""
	fraudEvalReportCache.mu.Unlock()

	snap, err := (&Service{}).fraudMLSnapshot(t.Context())
	require.NoError(t, err)
	require.Equal(t, "proxy", snap.LabelMethod)
	require.Equal(t, "2026-02-01T10:00:00Z", snap.EvalGeneratedAt)
	require.True(t, snap.EvalStale)
	require.Equal(t, "eval_stale", snap.EvalStatus)
}
