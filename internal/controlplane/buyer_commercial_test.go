package controlplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCampaignUtilizationPct(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 50, campaignUtilizationPct(50_000_000, 100_000_000), 0.01)
	assert.Equal(t, float64(0), campaignUtilizationPct(10, 0))
}

func TestPacingDriftPct(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 65, pacingDriftPct(500, "ACTIVE"), 0.01)
	assert.Equal(t, float64(0), pacingDriftPct(500, "PAUSED"))
}

func TestOverspendRisk(t *testing.T) {
	t.Parallel()
	assert.True(t, overspendRisk(92, "even", "ACTIVE"))
	assert.True(t, overspendRisk(80, "asap", "ACTIVE"))
	assert.False(t, overspendRisk(50, "even", "ACTIVE"))
	assert.False(t, overspendRisk(95, "even", "PAUSED"))
}

func TestPausedUnderDelivery(t *testing.T) {
	t.Parallel()
	assert.True(t, pausedUnderDelivery(500, 1_000_000))
	assert.False(t, pausedUnderDelivery(500, 0))
	assert.False(t, pausedUnderDelivery(10_000, 1_000_000))
}
