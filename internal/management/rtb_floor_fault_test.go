package management

import (
	"context"
	"espx/pkg/faultproof"
	"strconv"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_floor_optimizer_dry_run(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		BidFloorWinRateLow:  0.05,
		BidFloorWinRateHigh: 0.25,
		BidFloorAdjustPct:   10,
		BidFloorMinMicro:    1000,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Fault Floor", 1_000_000, "USD"))
	_, err := svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     "fault-floor-deal",
		FloorMicro: 150_000,
		CustomerID: customerID.String(),
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)

	_, err = svc.RunFloorOptimizer(ctx)
	require.NoError(t, err)

	result, err := svc.ApplyRtbFloorSuggestions(ctx, true, nil)
	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, 0, result.OutboxRows)

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount)

	faultproof.Log(t, "floor_optimizer_dry_run", map[string]string{
		"outbox_rows": strconv.Itoa(outboxCount),
		"dry_run":     "true",
	})
}

func TestApplyRtbFloorSuggestions_liveWritesRedisAndOutbox(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		BidFloorWinRateLow:  0.05,
		BidFloorWinRateHigh: 0.25,
		BidFloorAdjustPct:   10,
		BidFloorMinMicro:    1000,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Floor Live", 1_000_000, "USD"))
	_, err := svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     "live-deal",
		FloorMicro: 200_000,
		CustomerID: customerID.String(),
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)

	_, err = svc.RunFloorOptimizer(ctx)
	require.NoError(t, err)

	result, err := svc.ApplyRtbFloorSuggestions(ctx, false, nil)
	require.NoError(t, err)
	assert.False(t, result.DryRun)
	assert.Equal(t, 1, result.Applied)
	assert.Equal(t, 1, result.OutboxRows)

	val, err := rdb.Get(ctx, ingestion.RtbFloorRedisKeyPrefix+"live-deal").Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(200_000), val)

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'RELOAD_RTB_CATALOG'`).Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount)
}
