package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fraud silent reject outbox (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()
	worker := NewOutboxWorker(svc)
	ctx := context.Background()

	campaignID := uuid.New()
	ip := "203.0.113.88"
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, budget_limit, silent_reject_enabled)
		VALUES ($1, 'silent-reject-outbox', 'ACTIVE', 1000000, false)
	`, campaignID)
	require.NoError(t, err)

	payload, err := coldpath.MarshalOutbox(FraudThreatPayload{
		Action:     "silent_reject",
		IP:         ip,
		CampaignID: campaignID.String(),
		Score:      75,
		TTLSeconds: 300,
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		VALUES ('ML_SILENT_REJECT', $1, 'PENDING')`, payload)
	require.NoError(t, err)

	require.NoError(t, worker.ProcessOutbox(ctx))

	isMember, err := redisClient.SIsMember(ctx, "blacklist:fraud", ip).Result()
	require.NoError(t, err)
	assert.True(t, isMember)

	var silentReject bool
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT silent_reject_enabled FROM campaigns WHERE id = $1", campaignID).Scan(&silentReject))
	assert.False(t, silentReject, "ML_SILENT_REJECT must not flip campaign silent_reject_enabled")
}
