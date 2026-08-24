package controlplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcEdgeParityDivergence_exactMatch(t *testing.T) {
	assert.Equal(t, 0.0, calcEdgeParityDivergence(1000, 1000))
}

func TestCalcEdgeParityDivergence_belowAlertThreshold(t *testing.T) {
	got := calcEdgeParityDivergence(1000, 960)
	assert.InDelta(t, 0.04, got, 0.0001)
	assert.False(t, got > edgeParityDivergenceAlert)
}

func TestCalcEdgeParityDivergence_aboveAlertThreshold(t *testing.T) {
	got := calcEdgeParityDivergence(1000, 900)
	assert.InDelta(t, 0.1, got, 0.0001)
	assert.True(t, got > edgeParityDivergenceAlert)
}

func TestCalcEdgeParityDivergence_zeroEdgeIngress(t *testing.T) {
	assert.Equal(t, 0.0, calcEdgeParityDivergence(0, 0))
	assert.Equal(t, 1.0, calcEdgeParityDivergence(0, 10))
}

func TestSumEdgeBlocked(t *testing.T) {
	assert.Equal(t, uint64(12), sumEdgeBlocked(map[string]uint64{
		"ip_blacklist": 7,
		"campaign_rl":  5,
	}))
}

func TestShardMismatchHint(t *testing.T) {
	assert.Equal(t, "blacklist_stale_with_edge_blocks", shardMismatchHint(3, 1))
	assert.Equal(t, "blacklist_stale", shardMismatchHint(0, 2))
	assert.Empty(t, shardMismatchHint(0, 0))
}
