package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_MLBlacklistFastLaneDuringBacklog_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: ml blacklist fast lane during outbox backlog (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb1, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	var endpoint string
	switch client := rdb1.(type) {
	case *redis.Client:
		endpoint = client.Options().Addr
	default:
		t.Fatalf("unexpected redis client type")
	}

	rdb2 := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{endpoint}, DB: 1})
	defer rdb2.Close()
	rdb3 := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{endpoint}, DB: 2})
	defer rdb3.Close()

	var publishCount atomic.Int64
	wrap := func(rdb redis.UniversalClient) redis.UniversalClient {
		return &quarantinePublishCounter{UniversalClient: rdb, count: &publishCount}
	}

	ctx := context.Background()
	svc := newBareService(t, pool, []redis.UniversalClient{
		wrap(rdb1), wrap(rdb2), wrap(rdb3),
	}, &config.Config{})
	worker := NewOutboxWorker(svc)

	const pacingBacklog = 500
	pacingPayload, err := json.Marshal(campaignPacingPayload{
		CampaignID: uuid.New().String(),
		PacingMode: "even",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		SELECT 'UPDATE_CAMPAIGN_PACING', $1, 'PENDING'
		FROM generate_series(1, $2)`, pacingPayload, pacingBacklog)
	require.NoError(t, err)

	campaignID := uuid.New()
	mlIP := "203.0.113.77"
	require.NoError(t, svc.EnqueueFraudThreat(ctx, "blacklist", mlIP, campaignID.String(), 95, 0, 3600))

	var mlEventID int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT id FROM outbox_events
		WHERE event_type = 'ML_BLACKLIST_ADD' AND status = 'PENDING'
		ORDER BY id DESC LIMIT 1`).Scan(&mlEventID))

	processed, err := worker.ProcessOutboxWithCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	var mlStatus string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_events WHERE id = $1`, mlEventID).Scan(&mlStatus))
	assert.Equal(t, "PROCESSED", mlStatus)

	var pendingPacing int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events
		WHERE event_type = 'UPDATE_CAMPAIGN_PACING' AND status = 'PENDING'`).Scan(&pendingPacing))
	assert.Equal(t, pacingBacklog, pendingPacing)

	for _, rdb := range []redis.UniversalClient{rdb1, rdb2, rdb3} {
		ok, getErr := rdb.SIsMember(ctx, "blacklist:fraud", mlIP).Result()
		require.NoError(t, getErr)
		require.True(t, ok)
	}

	const shardCount = 3
	require.Equal(t, int64(shardCount), publishCount.Load(),
		"fraud:quarantine publish must be once per shard during backlog")

	faultproof.Log(t, "ml_blacklist_fast_lane_backlog", map[string]string{
		"subsystem":       "management_outbox",
		"pacing_backlog":  fmt.Sprintf("%d", pacingBacklog),
		"ml_processed":    "true",
		"quarantine_shards": fmt.Sprintf("%d", shardCount),
	})
}
