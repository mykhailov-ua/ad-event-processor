package outbox_test

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_FraudBoostPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
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

	svc := controlplane.NewService(context.Background(), pool, []redis.UniversalClient{rdb1, rdb2, rdb3}, nil, nil)
	defer svc.Close()
	worker := outbox.NewWorker(svc)
	ctx := context.Background()

	campID := uuid.New()
	payload, err := coldpath.MarshalOutbox(outbox.FraudThreatPayload{
		Action:     "boost",
		IP:         "203.0.113.70",
		CampaignID: campID.String(),
		Score:      45,
		Boost:      45,
		TTLSeconds: 300,
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		VALUES ('ML_SCORE_BOOST', $1, 'PENDING')`, payload)
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, worker.ProcessOutbox(ctx))
	elapsed := time.Since(start)

	key := "ml:score:boost:" + campID.String()
	for i, redisClient := range []redis.UniversalClient{rdb1, rdb2, rdb3} {
		val, getErr := redisClient.Get(ctx, key).Int()
		require.NoError(t, getErr, "shard %d missing boost key", i)
		assert.Equal(t, 45, val, "shard %d boost mismatch", i)
	}

	require.Less(t, elapsed, 30*time.Second, "boost propagation must complete within outbox lag SLA")

	faultproof.Log(t, "ml_boost_propagation", map[string]string{
		"subsystem":   "management",
		"shards":      "3",
		"elapsed_ms":  elapsed.String(),
		"sla_seconds": "30",
		"consistent":  "true",
	})
}

func TestFault_FraudSilentRejectAddsBlacklistNotCampaignFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fraud silent reject outbox (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := controlplane.NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()
	worker := outbox.NewWorker(svc)
	ctx := context.Background()

	campaignID := uuid.New()
	ip := "203.0.113.88"
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, budget_limit, silent_reject_enabled)
		VALUES ($1, 'silent-reject-outbox', 'ACTIVE', 1000000, false)
	`, campaignID)
	require.NoError(t, err)

	payload, err := coldpath.MarshalOutbox(outbox.FraudThreatPayload{
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
