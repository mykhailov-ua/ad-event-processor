package flow

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProportionalWeights_epc(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {Clicks: 100, Payout: 50},
		b: {Clicks: 100, Payout: 10},
	}
	weights := ProportionalWeights(BanditObjectiveEPC, stats, []uuid.UUID{a, b}, 0)
	require.Len(t, weights, 2)
	assert.Greater(t, weights[a], weights[b])
	assert.Greater(t, weights[b], int32(0))
}

func TestProportionalWeights_revenue(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {Clicks: 10, Payout: 200},
		b: {Clicks: 10, Payout: 50},
	}
	weights := ProportionalWeights(BanditObjectiveRevenue, stats, []uuid.UUID{a, b}, 0)
	require.Len(t, weights, 2)
	assert.Greater(t, weights[a], weights[b])
}

func TestProportionalWeights_zeroClicks(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {Clicks: 0, Payout: 0},
		b: {Clicks: 0, Payout: 0},
	}
	assert.Nil(t, ProportionalWeights(BanditObjectiveEPC, stats, []uuid.UUID{a, b}, 0))
}

func TestProportionalWeights_singleArm(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {Clicks: 100, Payout: 50},
		b: {Clicks: 0, Payout: 0},
	}
	assert.Nil(t, ProportionalWeights(BanditObjectiveEPC, stats, []uuid.UUID{a, b}, 0))
}

func TestProportionalWeights_tie(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {Clicks: 100, Payout: 50},
		b: {Clicks: 100, Payout: 50},
	}
	weights := ProportionalWeights(BanditObjectiveEPC, stats, []uuid.UUID{a, b}, 0)
	require.Len(t, weights, 2)
	assert.Equal(t, weights[a], weights[b])
}

func TestProportionalWeights_roi(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	b := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		a: {SpendMicro: 2_000_000, Payout: 6},
		b: {SpendMicro: 2_000_000, Payout: 3},
	}
	weights := ProportionalWeights(BanditObjectiveROI, stats, []uuid.UUID{a, b}, 1_000_000)
	require.Len(t, weights, 2)
	assert.Greater(t, weights[a], weights[b])
}

func TestProportionalWeights_holdout_invertedSort(t *testing.T) {
	t.Parallel()
	best := uuid.New()
	worst := uuid.New()
	stats := map[uuid.UUID]EntityBanditStat{
		best:  {Clicks: 200, Payout: 400},
		worst: {Clicks: 200, Payout: 40},
	}
	weights := ProportionalWeights(BanditObjectiveEPC, stats, []uuid.UUID{best, worst}, 0)
	require.NotNil(t, weights)
	assert.Greater(t, weights[best], weights[worst], "holdout: higher EPC arm must get higher weight")
}

func TestClampProposedWeight_respectsDelta(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int32(60), clampProposedWeight(50, 100, 20))
	assert.Equal(t, int32(40), clampProposedWeight(50, 10, 20))
}
