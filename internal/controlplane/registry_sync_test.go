package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := "test:campaign:updates"
	queries := db.New(pool)
	registry := testutil.NewWatchingRegistry(t, queries, redisClient, channel)

	cfg := &config.Config{
		CampaignUpdateChannel: channel,
	}
	cfg.Lifecycle.WaitTimeoutMs = 1
	sharder := domain.NewJumpHashSharder(1)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, sharder, cfg)
	defer svc.Close()

	customerID := uuid.New()
	_ = svc.CreateCustomer(ctx, customerID, "Sync db.User", 1_000_000_000, "USD")

	campaignID, err := svc.CreateCampaign(ctx, testCampaignSpec(customerID, "Sync Camp", 100_000_000, "idemp-sync"))
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return registry.Exists(campaignID)
	}, 2*time.Second, 100*time.Millisecond)

	err = svc.CancelCampaign(ctx, campaignID, "Test Sync")
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return !registry.Exists(campaignID)
	}, 2*time.Second, 100*time.Millisecond)
}
