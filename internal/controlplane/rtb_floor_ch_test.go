package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryClickHousePlacementFloorBuckets(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	conn, cleanupCH := setupClickHouseStatsTest(t)
	defer cleanupCH()

	ctx := context.Background()
	require.NoError(t, conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS rtb_deal_outcomes (
 deal_id LowCardinality(String),
 outcome UInt8,
 floor_micro Int64,
 created_at DateTime64(3, 'UTC')
) ENGINE = MergeTree()
ORDER BY (deal_id, created_at)`))

	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO rtb_deal_outcomes (deal_id, outcome, floor_micro, created_at) VALUES
		('bucket-deal', 1, 100000, now() - INTERVAL 1 HOUR),
		('bucket-deal', 0, 100000, now() - INTERVAL 1 HOUR)`))
	for range 12 {
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO rtb_deal_outcomes (deal_id, outcome, floor_micro, created_at)
			VALUES ('bucket-deal', 1, 110000, now() - INTERVAL 2 HOUR)`))
	}

	rows, err := conn.Query(ctx, `
SELECT
 deal_id,
 intDiv(floor_micro, 10000) * 10000 AS floor_bucket_micro,
 countIf(outcome = 1) AS wins,
 count() AS sample_n
FROM rtb_deal_outcomes
WHERE created_at >= now() - INTERVAL 168 HOUR
GROUP BY deal_id, floor_bucket_micro`)
	require.NoError(t, err)
	defer rows.Close()

	var bucket110 uint64
	for rows.Next() {
		var dealID string
		var floorBucket int64
		var wins, sampleN uint64
		require.NoError(t, rows.Scan(&dealID, &floorBucket, &wins, &sampleN))
		if dealID == "bucket-deal" && floorBucket == 110_000 {
			bucket110 = sampleN
			assert.Equal(t, sampleN, wins)
		}
	}
	require.NoError(t, rows.Err())
	assert.GreaterOrEqual(t, bucket110, uint64(10))
}

func TestRunFloorOptimizer_persistsSuggestions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{
		BidFloorWinRateLow:  0.05,
		BidFloorWinRateHigh: 0.25,
		BidFloorAdjustPct:   10,
		BidFloorMinMicro:    1000,
	}
	svc := newBareService(t, pool, nil, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Floor PG", 1_000_000, "USD"))
	_, err := svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     "pg-deal",
		FloorMicro: 200_000,
		CustomerID: customerID.String(),
	})
	require.NoError(t, err)

	n, err := svc.RunFloorOptimizer(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	rows, err := db.New(pool).ListRtbFloorSuggestions(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "pg-deal", rows[0].PlacementID)
	assert.Equal(t, int64(200_000), rows[0].SuggestedFloorMicro)
}

func TestApplyRtbFloorSuggestions_dryRunWritesNoRedisOrOutbox(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		BidFloorWinRateLow:  0.05,
		BidFloorWinRateHigh: 0.25,
		BidFloorAdjustPct:   10,
		BidFloorMinMicro:    1000,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Floor Apply", 1_000_000, "USD"))
	_, err := svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     "apply-deal",
		FloorMicro: 200_000,
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
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 0, result.OutboxRows)
	assert.NotEmpty(t, result.Suggestions)

	_, err = redisClient.Get(ctx, "rtb:floor:apply-deal").Result()
	assert.Error(t, err)

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount)
}
