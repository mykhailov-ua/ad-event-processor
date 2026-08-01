package ingestion

import (
	"context"
	"fmt"
	"testing"

	"espx/internal/database"
	"espx/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestVPPIntegration_snapshotSyncAndFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	campID := uuid.New()
	key := fmt.Sprintf("campaign:%s:pacing", campID.String())
	require.NoError(t, rdb.Set(ctx, key, "0.0", 0).Err())

	sw := &SettingsWatcher{rdbs: []redis.UniversalClient{rdb}}
	sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: make(map[uuid.UUID]float32)})
	sw.syncVPPRatios(ctx)
	require.Equal(t, float32(0.0), sw.GetVPPRatio(campID))

	reg := NewRegistry(nil)
	reg.Add(campID, uuid.New(), nil, "", domain.PacingModeVpp, 10_000_000, "UTC", 0, 0, nil)

	filter := NewVPPFilter(reg, sw)
	err := filter.Check(ctx, &domain.Event{CampaignID: campID})
	require.ErrorIs(t, err, ErrPacingExhausted)

	require.NoError(t, rdb.Set(ctx, key, "1.0", 0).Err())
	sw.syncVPPRatios(ctx)
	require.Equal(t, float32(1.0), sw.GetVPPRatio(campID))
	require.NoError(t, filter.Check(ctx, &domain.Event{CampaignID: campID}))
}
