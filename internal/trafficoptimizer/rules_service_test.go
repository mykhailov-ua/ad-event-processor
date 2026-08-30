package trafficoptimizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulesService_buildRuleParams_crPreset(t *testing.T) {
	t.Parallel()
	svc := &RulesService{EvalFloorMinutes: func() int { return 15 }}
	req, err := ApplyPreset(UpsertRuleRequest{
		PresetKey:  "cr_best_performer",
		CustomerID: "00000000-0000-4000-8000-000000000001",
		Scope:      ScopeLander,
		Enabled:    true,
	})
	require.NoError(t, err)
	params, err := svc.buildRuleParams(req)
	require.NoError(t, err)
	assert.Equal(t, ObjectiveCR, params.Objective)
	assert.Equal(t, AlgorithmThompson, params.Algorithm)
	assert.Equal(t, ScopeLander, params.Scope)
}

func TestRulesService_buildRuleParams_epcPreset(t *testing.T) {
	t.Parallel()
	svc := &RulesService{EvalFloorMinutes: func() int { return 15 }}
	req, err := ApplyPreset(UpsertRuleRequest{
		PresetKey:  "epc_best_performer",
		CustomerID: "00000000-0000-4000-8000-000000000001",
		Scope:      ScopeLander,
		Enabled:    true,
	})
	require.NoError(t, err)
	params, err := svc.buildRuleParams(req)
	require.NoError(t, err)
	assert.Equal(t, ObjectiveEPC, params.Objective)
	assert.Equal(t, AlgorithmProportional, params.Algorithm)
}

func TestRulesService_buildRuleParams_roiPreset(t *testing.T) {
	t.Parallel()
	svc := &RulesService{EvalFloorMinutes: func() int { return 15 }}
	req, err := ApplyPreset(UpsertRuleRequest{
		PresetKey:  "roi_best_performer",
		CustomerID: "00000000-0000-4000-8000-000000000001",
		BrandID:    "00000000-0000-4000-8000-000000000002",
		Scope:      ScopeCreative,
		Enabled:    true,
	})
	require.NoError(t, err)
	params, err := svc.buildRuleParams(req)
	require.NoError(t, err)
	assert.Equal(t, ObjectiveROI, params.Objective)
	assert.Equal(t, ScopeCreative, params.Scope)
	assert.Equal(t, AlgorithmProportional, params.Algorithm)
}

func TestRulesService_buildRuleParams_rejectsBrandOnLanderScope(t *testing.T) {
	t.Parallel()
	svc := &RulesService{EvalFloorMinutes: func() int { return 15 }}
	_, err := svc.buildRuleParams(UpsertRuleRequest{
		CustomerID: "00000000-0000-4000-8000-000000000001",
		BrandID:    "00000000-0000-4000-8000-000000000002",
		Name:       "bad",
		Scope:      ScopeLander,
		Objective:  ObjectiveCR,
		Algorithm:  AlgorithmThompson,
		Enabled:    true,
	})
	require.Error(t, err)
}
