package flow

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttributeCreativeBanditStats_directCreative(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	chStats := map[uuid.UUID]CreativeBanditStat{
		a: {Impressions: 2000, Clicks: 100, SpendMicro: 2_000_000, Payout: 10},
		b: {Impressions: 2000, Clicks: 100, SpendMicro: 2_000_000, Payout: 3},
	}
	stats := AttributeCreativeBanditStats([]uuid.UUID{a, b}, nil, chStats, 1000)
	require.Len(t, stats, 2)
	weights := ProportionalWeights(BanditObjectiveROI, stats, []uuid.UUID{a, b}, 1_000_000)
	require.Len(t, weights, 2)
	assert.Greater(t, weights[a], weights[b])
}

func TestApplyCreativeProportionalWeights_clampsDelta(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	current := map[uuid.UUID]int32{a: 50, b: 50}
	stats := map[uuid.UUID]EntityBanditStat{
		a: {SpendMicro: 2_000_000, Payout: 10},
		b: {SpendMicro: 2_000_000, Payout: 3},
	}
	out := ApplyCreativeProportionalWeights([]uuid.UUID{a, b}, current, stats, BanditApplyConfig{
		Objective:         BanditObjectiveROI,
		MinSpendMicro:     1_000_000,
		MaxWeightDeltaPct: 10,
	})
	require.NotEmpty(t, out)
	assert.LessOrEqual(t, out[a], int32(55))
}
