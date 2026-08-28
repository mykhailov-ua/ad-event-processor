package reports

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSilentRejectImpressionFunnelQuery_joinsFraudEvents_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, silentRejectImpressionFunnelQuery, "LEFT JOIN fraud_events AS fe")
	require.Contains(t, silentRejectImpressionFunnelQuery, "silent_reject_event = 1")
	require.Contains(t, silentRejectImpressionFunnelQuery, "event_type = 'impression'")
}

func TestIvtBySourceQuery_joinsImpressionFraud_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, ivtBySourceQuery, "FROM impressions AS i")
	require.Contains(t, ivtBySourceQuery, "LEFT JOIN fraud_events AS f")
	assert.Equal(t, 2, strings.Count(ivtBySourceQuery, "LEFT JOIN fraud_events"))
}

func TestCalcSilentRejectImpressionRate_holdout(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 0.2, calcSilentRejectImpressionRate(20, 80), 0.0001)
	assert.Equal(t, float64(0), calcSilentRejectImpressionRate(0, 0))
	assert.InDelta(t, 1.0, calcSilentRejectImpressionRate(5, 0), 0.0001)
}
