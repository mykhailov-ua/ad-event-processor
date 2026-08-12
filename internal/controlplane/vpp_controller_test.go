package controlplane

import (
	"context"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunVPPPacingController_writesRedisRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		PacingToleranceMargin: 0.0,
	}
	sharder := domain.NewStaticSlotSharder(1)
	svc := NewService(pool, []redis.UniversalClient{rdb}, sharder, cfg)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "VPP Customer", 1_000_000_000, "USD"))

	campaignID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		Name:             "VPP Campaign",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       string(db.PacingModeTypeVPP),
		DailyBudgetMicro: 100_000_000,
		Timezone:         "UTC",
		FreqWindow:       86400,
		IdempotencyKey:   "vpp-controller-idem",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE campaigns
		SET pacing_mode = 'VPP', current_spend = 80000000, daily_budget = 100000000
		WHERE id = $1`, domain.ToUUID(campaignID))
	require.NoError(t, err)

	require.NoError(t, svc.RunVPPPacingController(ctx))

	raw, err := rdb.Get(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	ratio, err := strconv.ParseFloat(raw, 32)
	require.NoError(t, err)
	assert.Less(t, ratio, 1.0)
	assert.Greater(t, ratio, 0.0)
}

func TestRunVPPPacingController_onPaceWritesFullRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{PacingToleranceMargin: 0.15}
	svc := NewService(pool, []redis.UniversalClient{rdb}, domain.NewStaticSlotSharder(1), cfg)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "VPP On-pace", 1_000_000_000, "USD"))

	campaignID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		Name:             "VPP On Pace",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       string(db.PacingModeTypeVPP),
		DailyBudgetMicro: 100_000_000,
		Timezone:         "UTC",
		FreqWindow:       86400,
		IdempotencyKey:   "vpp-onpace-idem",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE campaigns
		SET pacing_mode = 'VPP', current_spend = 1000000, daily_budget = 100000000
		WHERE id = $1`, domain.ToUUID(campaignID))
	require.NoError(t, err)

	require.NoError(t, svc.RunVPPPacingController(ctx))

	raw, err := rdb.Get(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	ratio, err := strconv.ParseFloat(raw, 32)
	require.NoError(t, err)
	assert.Equal(t, 1.0, ratio)
}

func TestRunVPPPacingController_skipsNonVPP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(pool, []redis.UniversalClient{rdb}, domain.NewStaticSlotSharder(1), &config.Config{})
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "EVEN Customer", 1_000_000_000, "USD"))

	campaignID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		Name:             "EVEN Campaign",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       string(db.PacingModeTypeEVEN),
		DailyBudgetMicro: 100_000_000,
		Timezone:         "UTC",
		FreqWindow:       86400,
		IdempotencyKey:   "vpp-skip-idem",
	})
	require.NoError(t, err)

	require.NoError(t, svc.RunVPPPacingController(ctx))

	exists, err := rdb.Exists(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}
