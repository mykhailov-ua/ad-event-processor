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

func TestUnifiedFilter_skipBudgetDebit_preservesRedisBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	f := newRealRedisUnifiedFilter(t, redisClient)
	f.SetSkipBudgetDebit(true)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	seedCampaignBudget(t, ctx, redisClient, campID)
	before, err := redisClient.Get(ctx, "budget:campaign:"+campID.String()).Int64()
	require.NoError(t, err)

	evt := &domain.Event{
		Type:       "click",
		CampaignID: campID,
		IP:         "1.1.1.1",
		ClickID:    uuid.NewString(),
	}
	require.NoError(t, f.Check(attachFilterDeadline(ctx, time.Second), evt))

	after, err := redisClient.Get(ctx, "budget:campaign:"+campID.String()).Int64()
	require.NoError(t, err)
	assert.Equal(t, before, after, "skip_budget must not debit Redis campaign budget")
}
