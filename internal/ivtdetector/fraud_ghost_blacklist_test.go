package ivtdetector

import (
	"context"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"
	"espx/internal/management"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMLGhostAndBlacklist_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()

	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, budget_limit, fraud_threshold_pass, fraud_threshold_suspect, fraud_threshold_block, ghost_ivt_enabled)
		VALUES ($1, 'Test Campaign', 'ACTIVE', 1000000000, 20, 50, 90, true)
	`, campaignID)
	require.NoError(t, err)

	cfg := &config.Config{}
	sharder := ingestion.NewStaticSlotSharder(1)
	svc := management.NewService(pool, []redis.UniversalClient{rdb}, sharder, cfg)

	worker := management.NewOutboxWorker(svc)

	err = svc.EnqueueFraudThreat(ctx, management.FraudThreatPayload{
		Action:     "ghost",
		CampaignID: campaignID.String(),
		IP:         "1.1.1.1",
		Score:      75.0,
		TTLSeconds: 300,
	})
	require.NoError(t, err)

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Greater(t, processed, 0)

	_, err = pool.Exec(ctx, "UPDATE campaigns SET ghost_ivt_enabled = FALSE WHERE id = $1", campaignID)
	require.NoError(t, err)

	err = svc.EnqueueFraudThreat(ctx, management.FraudThreatPayload{
		Action:     "ghost",
		CampaignID: campaignID.String(),
		IP:         "1.1.1.1",
		Score:      75.0,
		TTLSeconds: 300,
	})
	require.NoError(t, err)

	processed, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Greater(t, processed, 0)

	var ghostEnabled bool
	err = pool.QueryRow(ctx, "SELECT ghost_ivt_enabled FROM campaigns WHERE id = $1", campaignID).Scan(&ghostEnabled)
	require.NoError(t, err)
	assert.True(t, ghostEnabled)

	err = svc.EnqueueFraudThreat(ctx, management.FraudThreatPayload{
		Action:     "blacklist",
		CampaignID: campaignID.String(),
		IP:         "9.9.9.9",
		Score:      95.0,
		TTLSeconds: 3600,
	})
	require.NoError(t, err)

	processed, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Greater(t, processed, 0)

	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ip_blacklist WHERE ip = '9.9.9.9')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)

	err = svc.ApplyFraudScoringOverride(ctx, management.FraudScoringOverrideRequest{
		CampaignID: ptr(campaignID.String()),
	})
	require.NoError(t, err)

	processed, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Greater(t, processed, 0)

	err = svc.ApplyFraudScoringOverride(ctx, management.FraudScoringOverrideRequest{
		IP: ptr("9.9.9.9"),
	})
	require.NoError(t, err)

	processed, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Greater(t, processed, 0)

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ip_blacklist WHERE ip = '9.9.9.9')").Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}

func ptr[T any](v T) *T {
	return &v
}
