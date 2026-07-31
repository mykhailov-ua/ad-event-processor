package fraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResidentialProxySignal(t *testing.T) {
	row := FeatureRow{
		Events:      275,
		Clicks:      4,
		UniqueUsers: 32,
		UniqueUAs:   11,
	}
	assert.True(t, ResidentialProxySignal(row))

	organic := FeatureRow{
		Events:      1000,
		Clicks:      120,
		UniqueUsers: 80,
		UniqueUAs:   70,
	}
	assert.False(t, ResidentialProxySignal(organic))
}

func TestAdjustProbabilityFPGuard(t *testing.T) {
	row := FeatureRow{
		Events:      1139,
		Clicks:      1046,
		UniqueUsers: 3,
		UniqueUAs:   3,
	}
	adjusted, proxy, structural, fpGuard := AdjustProbability(row, 0.91)
	require.False(t, proxy)
	require.True(t, structural)
	require.False(t, fpGuard)
	assert.InDelta(t, 0.91, adjusted, 1e-9)

	grey := FeatureRow{
		Events:      500,
		Clicks:      10,
		UniqueUsers: 200,
		UniqueUAs:   120,
	}
	adjusted, proxy, structural, fpGuard = AdjustProbability(grey, 0.91)
	require.False(t, proxy)
	require.False(t, structural)
	require.True(t, fpGuard)
	assert.InDelta(t, GetPolicyConfig().FPGuardCap, adjusted, 1e-9)
}

func TestDecideResidentialProxyFloor(t *testing.T) {
	row := FeatureRow{
		Events:      400,
		Clicks:      6,
		UniqueUsers: 40,
		UniqueUAs:   8,
	}
	decision := Decide(row, 0.35)
	assert.True(t, decision.ResidentialProxy)
	assert.GreaterOrEqual(t, decision.Score, 62)
	assert.Equal(t, FraudTierIVT, decision.Tier)
}
