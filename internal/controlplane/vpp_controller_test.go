package controlplane

import (
	"context"
	"strconv"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineWriteVPPRatios_batchesPerShard(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	svc := &Service{
		redisShards: []redis.UniversalClient{redisClient},
		sharder:     domain.NewStaticSlotSharder(1),
	}
	c1, c2 := uuid.New(), uuid.New()
	writes := map[int][]vppRatioWrite{
		0: {
			{campaignID: c1, ratio: 0.75},
			{campaignID: c2, ratio: 1.0},
		},
	}
	require.NoError(t, svc.pipelineWriteVPPRatios(ctx, writes))

	raw1, err := redisClient.Get(ctx, vppPacingRedisKey(c1)).Result()
	require.NoError(t, err)
	assert.Equal(t, "0.7500", raw1)
	raw2, err := redisClient.Get(ctx, vppPacingRedisKey(c2)).Result()
	require.NoError(t, err)
	assert.Equal(t, "1.0000", raw2)
}

func TestRunVPPPacingController_writesRedisRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		PacingToleranceMargin: 0.0,
	}
	sharder := domain.NewStaticSlotSharder(1)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, sharder, cfg)
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

	raw, err := redisClient.Get(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	ratio, err := strconv.ParseFloat(raw, 32)
	require.NoError(t, err)
	assert.Less(t, ratio, 1.0)
	assert.Greater(t, ratio, 0.0)
}

func TestRunVPPPacingController_onPaceWritesFullRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{PacingToleranceMargin: 0.15}
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, domain.NewStaticSlotSharder(1), cfg)
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

	raw, err := redisClient.Get(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	ratio, err := strconv.ParseFloat(raw, 32)
	require.NoError(t, err)
	assert.Equal(t, 1.0, ratio)
}

func TestRunVPPPacingController_skipsNonVPP(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, domain.NewStaticSlotSharder(1), &config.Config{})
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

	exists, err := redisClient.Exists(ctx, vppPacingRedisKey(campaignID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}
