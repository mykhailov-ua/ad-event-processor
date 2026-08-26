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

func TestFilterEngine_shadowSkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: shadow must skip unified Lua budget debit (run make test-integration)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	unified := newRealRedisUnifiedFilter(t, redisClient)
	require.NoError(t, unified.PreloadScripts(ctx))

	table := NewDCASNTable()
	table.Publish(buildDCASNSnapshot(map[uint32]struct{}{16509: {}}, 1))
	geo := &MockGeoProvider{ASN: map[string]uint32{"54.230.17.9": 16509}}
	fraud := NewFraudFilter(geo)
	fraud.ConfigureDCASN(table, geo, -1)

	engine := NewFilterEngine(0, fraud, unified)
	engine.SetRegistry(&mockRegistry{})

	campID := uuid.New()
	seedCampaignBudget(t, ctx, redisClient, campID)
	regCamp, ok := (&mockRegistry{}).GetCampaign(campID)
	require.True(t, ok)

	evt := &domain.Event{
		Type:       "click",
		IP:         "54.230.17.9",
		UserID:     "user-shadow",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}

	beforeSpend, err := redisClient.Get(ctx, regCamp.BudgetCampaignKey).Int64()
	require.NoError(t, err)

	err = engine.Check(attachFilterDeadline(ctx, time.Second), evt)
	require.NoError(t, err)
	assert.True(t, evt.ShadowEvent)

	afterSpend, err := redisClient.Get(ctx, regCamp.BudgetCampaignKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, beforeSpend, afterSpend, "L2 shadow must not debit campaign budget via unified Lua")
}
