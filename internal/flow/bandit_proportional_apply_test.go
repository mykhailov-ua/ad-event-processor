package flow

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFlowBanditProportional_epcLanders(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	landerA := uuid.New()
	landerB := uuid.New()
	raw, err := json.Marshal([]banditPathJSON{{
		Weight: 100,
		Landers: []banditLanderJSON{
			{LanderID: landerA, Weight: 50},
			{LanderID: landerB, Weight: 50},
		},
	}})
	require.NoError(t, err)

	stats := map[uuid.UUID]map[uuid.UUID]EntityBanditStat{
		campID: {
			landerA: {Clicks: 200, Payout: 400},
			landerB: {Clicks: 200, Payout: 40},
		},
	}
	out, changed, err := ApplyFlowBanditProportional(raw, []uuid.UUID{campID}, stats, nil, BanditApplyConfig{
		Scope:     "lander",
		Objective: BanditObjectiveEPC,
		Algorithm: AlgorithmProportional,
		MinClicks: 100,
	})
	require.NoError(t, err)
	require.True(t, changed)

	var paths []banditPathJSON
	require.NoError(t, json.Unmarshal(out, &paths))
	require.Len(t, paths[0].Landers, 2)
	weightByID := map[uuid.UUID]int32{}
	for _, l := range paths[0].Landers {
		weightByID[l.LanderID] = l.Weight
	}
	assert.Greater(t, weightByID[landerA], weightByID[landerB])
}
