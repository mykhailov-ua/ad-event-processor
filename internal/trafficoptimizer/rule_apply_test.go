package trafficoptimizer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleSupported_epcAndRevenue(t *testing.T) {
	t.Parallel()
	require.True(t, RuleSupported(Rule{Scope: ScopeLander, Objective: ObjectiveCR, Algorithm: AlgorithmThompson}))
	require.True(t, RuleSupported(Rule{Scope: ScopeOffer, Objective: ObjectiveROI, Algorithm: AlgorithmProportional}))
	require.True(t, RuleSupported(Rule{
		Scope:     ScopeCreative,
		Objective: ObjectiveROI,
		Algorithm: AlgorithmProportional,
		HasBrand:  true,
		BrandID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
	}))
	require.False(t, RuleSupported(Rule{Scope: ScopeLander, Objective: ObjectiveEPC, Algorithm: AlgorithmThompson}))
	require.False(t, RuleSupported(Rule{Scope: ScopeCreative, Objective: ObjectiveCR, Algorithm: AlgorithmThompson}))
}

func TestRuleSupported_holdout_epcRequiresProportional(t *testing.T) {
	t.Parallel()
	assert.False(t, RuleSupported(Rule{
		Scope:     ScopeLander,
		Objective: ObjectiveEPC,
		Algorithm: AlgorithmThompson,
	}), "holdout: EPC with Thompson must not run")
}
