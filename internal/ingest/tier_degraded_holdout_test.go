package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Holdout: near-deadline Lua code 20 must not accept when fcap would block on full path.
func TestUnifiedFilter_TierDegraded_failClosed_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	reg := &benchWorstRegistry{}
	f := newRealRedisUnifiedFilter(t, redisClient)
	f.SetRegistry(reg)
	f.SetLuaFastPathEnabled(false)
	f.SetTTCMin(0)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	camp, ok := reg.GetCampaign(campID)
	require.True(t, ok)
	require.NoError(t, redisClient.Set(ctx, camp.BudgetCampaignKey, 9_000_000_000_000_000, 0).Err())
	require.NoError(t, redisClient.Set(ctx, camp.FcapKeyPrefix+"cap-user", 999, 0).Err())

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.99",
		UserID:     "cap-user",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	evt.FilterDeadlineMono = monotonicNano() + 500_000

	err := f.Check(ctx, evt)
	require.ErrorIs(t, err, ErrFilterTimeout, "holdout: degraded must not accept past fcap gate")

	fcapAfter, fcapErr := redisClient.Get(ctx, camp.FcapKeyPrefix+"cap-user").Int64()
	require.NoError(t, fcapErr)
	require.Equal(t, int64(999), fcapAfter, "holdout: degraded accept would increment fcap")
}
