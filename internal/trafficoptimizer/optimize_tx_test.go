package trafficoptimizer

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOptimizeHost struct{}

func (mockOptimizeHost) MABLookbackDays() int { return 90 }

func (mockOptimizeHost) QueryFlowBanditStats(context.Context, time.Time, time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	return nil, nil, nil
}

func TestApplyRuleTx_holdout_crScopeGate(t *testing.T) {
	t.Parallel()
	_, applied, err := ApplyRuleTx(
		context.Background(),
		nil,
		mockOptimizeHost{},
		Rule{Objective: ObjectiveCR, Algorithm: AlgorithmThompson, Scope: ScopeCreative},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.False(t, applied, "holdout: creative scope must not apply for CR rules")
}

func TestApplyRuleTx_holdout_epcThompsonRejected(t *testing.T) {
	t.Parallel()
	_, applied, err := ApplyRuleTx(
		context.Background(),
		nil,
		mockOptimizeHost{},
		Rule{Objective: ObjectiveEPC, Algorithm: AlgorithmThompson, Scope: ScopeLander},
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.False(t, applied, "holdout: EPC requires proportional algorithm")
}

func TestApplyRuleTx_requiresHost(t *testing.T) {
	t.Parallel()
	_, _, err := ApplyRuleTx(context.Background(), nil, nil, Rule{Objective: ObjectiveCR}, time.Now().UTC())
	require.Error(t, err)
}
