package trafficoptimizer

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeObjective_includesROI(t *testing.T) {
	t.Parallel()
	for _, objective := range []string{ObjectiveCR, ObjectiveEPC, ObjectiveRevenue, ObjectiveROI} {
		got, err := NormalizeObjective(objective)
		require.NoError(t, err)
		assert.Equal(t, objective, got)
	}
}

func TestValidateObjectiveAlgorithmPair(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateObjectiveAlgorithmPair(ObjectiveCR, AlgorithmThompson))
	require.NoError(t, ValidateObjectiveAlgorithmPair(ObjectiveEPC, AlgorithmProportional))
	require.NoError(t, ValidateObjectiveAlgorithmPair(ObjectiveROI, AlgorithmProportional))
	require.Error(t, ValidateObjectiveAlgorithmPair(ObjectiveEPC, AlgorithmThompson))
	require.Error(t, ValidateObjectiveAlgorithmPair(ObjectiveCR, AlgorithmProportional))
}

func TestValidateRuleTargets_creativeRequiresBrand(t *testing.T) {
	t.Parallel()
	require.Error(t, ValidateRuleTargets(ScopeCreative, "", "", ""))
	require.NoError(t, ValidateRuleTargets(ScopeCreative, "00000000-0000-4000-8000-000000000001", "", ""))
	require.Error(t, ValidateRuleTargets(ScopeLander, "00000000-0000-4000-8000-000000000001", "", ""))
}

func TestValidateMinClicks(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateMinClicks(100))
	require.Error(t, ValidateMinClicks(50))
}

func TestNormalizeEvalIntervalMinutes_floor(t *testing.T) {
	t.Parallel()
	_, err := NormalizeEvalIntervalMinutes(5, 15)
	require.Error(t, err)

	got, err := NormalizeEvalIntervalMinutes(15, 15)
	require.NoError(t, err)
	assert.Equal(t, 15, got)
}

func TestRuleDueForEval(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	last := now.Add(-20 * time.Minute)
	assert.True(t, RuleDueForEval(now, last, true, 15))
	assert.False(t, RuleDueForEval(now, last, true, 30))
	assert.True(t, RuleDueForEval(now, time.Time{}, false, 15))
}

func TestApplyActionHash_stable(t *testing.T) {
	t.Parallel()
	ruleID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	flowID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	end := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	h1 := ApplyActionHash(ruleID, flowID, end)
	h2 := ApplyActionHash(ruleID, flowID, end)
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, ApplyActionHash(ruleID, uuid.Nil, end))
}
