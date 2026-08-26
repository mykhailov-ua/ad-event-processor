package automation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThresholdBreached(t *testing.T) {
	t.Parallel()
	require.True(t, ThresholdBreached("lt", 5, 10))
	require.False(t, ThresholdBreached("lt", 15, 10))
	require.True(t, ThresholdBreached("gte", 10, 10))
}

func TestCalcROIPct(t *testing.T) {
	t.Parallel()
	require.Equal(t, float64(50), CalcROIPct(500_000, 1_000_000))
	require.Equal(t, float64(0), CalcROIPct(100, 0))
}

func TestCalcIVTRatePct_zeroClicks(t *testing.T) {
	t.Parallel()
	require.Equal(t, float64(0), CalcIVTRatePct(0, 10))
	require.Equal(t, float64(25), CalcIVTRatePct(100, 25))
}

func TestCalcSilentRejectRatePct_zeroClicks(t *testing.T) {
	t.Parallel()
	require.Equal(t, float64(0), CalcSilentRejectRatePct(0, 5))
	require.Equal(t, float64(15), CalcSilentRejectRatePct(100, 15))
}

func TestNormalizeMetric_fraudMetrics(t *testing.T) {
	t.Parallel()
	got, err := NormalizeMetric("ivt_rate")
	require.NoError(t, err)
	require.Equal(t, "fraud_reject_rate", got)
	got, err = NormalizeMetric("fraud_reject_rate")
	require.NoError(t, err)
	require.Equal(t, "fraud_reject_rate", got)
	for _, metric := range []string{"silent_reject_rate", "fraud_reject_count"} {
		got, err := NormalizeMetric(metric)
		require.NoError(t, err)
		require.Equal(t, metric, got)
	}
}
