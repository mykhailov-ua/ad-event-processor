package controlplane

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBandit_WorkerUpdatesWeights(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	paths := []flowBanditPathJSON{{
		Weight: 100,
		Landers: []flowBanditLanderJSON{
			{LanderID: landerA, Weight: 50},
			{LanderID: landerB, Weight: 50},
		},
	}}
	raw, err := json.Marshal(paths)
	require.NoError(t, err)

	landerStats := map[uuid.UUID]map[uuid.UUID]flowEntityStat{
		campID: {
			landerA: {clicks: 500, conversions: 50},
			landerB: {clicks: 500, conversions: 5},
		},
	}
	rng := rand.New(rand.NewSource(7))
	out, changed, err := applyFlowBanditThompson(raw, []uuid.UUID{campID}, landerStats, nil, rng)
	require.NoError(t, err)
	require.True(t, changed)
	require.NotEmpty(t, out)

	var updated []flowBanditPathJSON
	require.NoError(t, json.Unmarshal(out, &updated))
	require.Len(t, updated, 1)
	require.NotEqual(t, 50, updated[0].Landers[0].Weight)
	require.NotEqual(t, 50, updated[0].Landers[1].Weight)
}

func TestApplyFlowBanditThompson_skipsLowClicks(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	campID := uuid.New()
	raw, err := json.Marshal([]flowBanditPathJSON{{
		Weight: 100,
		Landers: []flowBanditLanderJSON{
			{LanderID: landerA, Weight: 50},
			{LanderID: landerB, Weight: 50},
		},
	}})
	require.NoError(t, err)
	stats := map[uuid.UUID]map[uuid.UUID]flowEntityStat{
		campID: {
			landerA: {clicks: 10, conversions: 1},
			landerB: {clicks: 10, conversions: 0},
		},
	}
	_, changed, err := applyFlowBanditThompson(raw, []uuid.UUID{campID}, stats, nil, rand.New(rand.NewSource(1)))
	require.NoError(t, err)
	require.False(t, changed)
}
