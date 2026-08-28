package ingestion

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterEngine_smokeSkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: smoke must skip unified Lua budget debit (run make test-integration)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	unified := newRealRedisUnifiedFilter(t, redisClient)
	require.NoError(t, unified.PreloadScripts(ctx))

	engine := NewFilterEngine(0, unified)
	engine.SetRegistry(&mockRegistry{})

	campID := uuid.New()
	seedCampaignBudget(t, ctx, redisClient, campID)
	regCamp, ok := (&mockRegistry{}).GetCampaign(campID)
	require.True(t, ok)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.9",
		UserID:     "user-smoke",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		SmokeEvent: true,
	}

	beforeSpend, err := redisClient.Get(ctx, regCamp.BudgetCampaignKey).Int64()
	require.NoError(t, err)

	err = engine.Check(attachFilterDeadline(ctx, time.Second), evt)
	require.NoError(t, err)
	assert.True(t, evt.SmokeEvent)

	afterSpend, err := redisClient.Get(ctx, regCamp.BudgetCampaignKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, beforeSpend, afterSpend, "smoke click must not debit campaign budget via unified Lua")
}
