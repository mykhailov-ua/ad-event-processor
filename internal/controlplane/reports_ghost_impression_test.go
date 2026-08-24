package controlplane

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGhostImpressionFunnelQuery_joinsFraudEvents_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, ghostImpressionFunnelQuery, "LEFT JOIN fraud_events AS fe")
	require.Contains(t, ghostImpressionFunnelQuery, "ghost_event = 1")
	require.Contains(t, ghostImpressionFunnelQuery, "event_type = 'impression'")
}

func TestIvtBySourceQuery_joinsImpressionFraud_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, ivtBySourceQuery, "FROM impressions AS i")
	require.Contains(t, ivtBySourceQuery, "LEFT JOIN fraud_events AS f")
	assert.Equal(t, 2, strings.Count(ivtBySourceQuery, "LEFT JOIN fraud_events"))
}

func TestCalcGhostImpressionRate_holdout(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 0.2, calcGhostImpressionRate(20, 80), 0.0001)
	assert.Equal(t, float64(0), calcGhostImpressionRate(0, 0))
	assert.InDelta(t, 1.0, calcGhostImpressionRate(5, 0), 0.0001)
}
