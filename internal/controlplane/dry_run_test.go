package controlplane

import (
	"context"
	"net/http"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRun_PauseCampaignNoSideEffects(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, domain.NewJumpHashSharder(1), nil)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "dry-run-advertiser", 5_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		Name:             "dry-run-pause",
		BudgetLimitMicro: 5_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		FreqWindow:       86400,
	})
	require.NoError(t, err)

	var outboxBefore int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxBefore))

	preview, err := svc.PreviewPauseCampaign(ctx, campID, "dry-run")
	require.NoError(t, err)
	assert.True(t, preview.DryRun)
	assert.Equal(t, "PAUSE_CAMPAIGN", preview.Action)

	camp, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypeACTIVE, camp.Status)

	var outboxAfter int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxAfter))
	assert.Equal(t, outboxBefore, outboxAfter)

	isMember, err := rdb.SIsMember(ctx, "blacklist:fraud", "10.0.0.2").Result()
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestDryRun_BlockIPNoSideEffects(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, domain.NewJumpHashSharder(1), nil)
	defer svc.Close()

	ctx := context.Background()
	ip := "198.51.100.77"

	preview, err := svc.PreviewBlockIP(ctx, ip, "fraud", nil)
	require.NoError(t, err)
	assert.True(t, preview.DryRun)

	var blacklistCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM ip_blacklist WHERE ip = $1`, ip).Scan(&blacklistCount))
	assert.Equal(t, 0, blacklistCount)

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_BLACKLIST'`).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount)

	isMember, err := rdb.SIsMember(ctx, "blacklist:fraud", ip).Result()
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestParseDryRun(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("POST", "/api/v1/selfserve/campaigns/x/pause?dry_run=1", http.NoBody)
	assert.True(t, ParseDryRun(req))

	req, _ = http.NewRequest("POST", "/api/v1/selfserve/campaigns/x/pause", http.NoBody)
	req.Header.Set("X-Dry-Run", "1")
	assert.True(t, ParseDryRun(req))

	req, _ = http.NewRequest("POST", "/api/v1/selfserve/campaigns/x/pause", http.NoBody)
	assert.False(t, ParseDryRun(req))
}
