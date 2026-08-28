package smartalerts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAlertThresholdBreached(t *testing.T) {
	t.Parallel()
	require.True(t, alertThresholdBreached("gt", 10, 5))
	require.False(t, alertThresholdBreached("gt", 5, 5))
	require.True(t, alertThresholdBreached("gte", 5, 5))
	require.True(t, alertThresholdBreached("lt", 3, 5))
	require.True(t, alertThresholdBreached("lte", 5, 5))
}

func TestAlertWindowBounds(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 13, 47, 30, 0, time.UTC)
	start, end := alertWindowBounds(now, 60)
	require.Equal(t, time.Date(2026, 8, 12, 12, 47, 0, 0, time.UTC), start)
	require.Equal(t, time.Date(2026, 8, 12, 13, 47, 0, 0, time.UTC), end)
}

func TestNormalizeAlertMetric(t *testing.T) {
	t.Parallel()
	m, err := normalizeAlertMetric(" ROI_PCT ")
	require.NoError(t, err)
	require.Equal(t, "roi_pct", m)
	_, err = normalizeAlertMetric("unknown")
	require.Error(t, err)
}

func TestClampAlertWindowMinutes(t *testing.T) {
	t.Parallel()
	require.Equal(t, 5, clampAlertWindowMinutes(1))
	require.Equal(t, 60, clampAlertWindowMinutes(60))
	require.Equal(t, 1440, clampAlertWindowMinutes(9999))
}
