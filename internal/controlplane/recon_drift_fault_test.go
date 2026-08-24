package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_ReconDriftRefill(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance) VALUES ($1, 'recon', 1000000)`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend)
		VALUES ($1, 'recon-camp', 'ACTIVE', $2, 10000000, 500000)`, campaignID, customerID)
	require.NoError(t, err)

	sharder := domain.NewStaticSlotSharder(1)
	cfg := &config.Config{ReconForceRefill: true}
	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, sharder, cfg)
	require.NotNil(t, svc)

	require.NoError(t, svc.forceRefillCampaignFromPG(ctx, campaignID, 500_000))

	remaining, err := rdb.Get(ctx, domain.BudgetCampaignKey(campaignID)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(9_500_000), remaining)

	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)

	faultproof.Log(t, "recon_drift_within_band", map[string]string{
		"fault":       "recon_drift_within_band",
		"campaign_id": campaignID.String(),
	})
}
