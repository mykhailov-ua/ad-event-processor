package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"

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

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb1, rdb2, rdb3}, nil, nil)
	defer svc.Close()
	worker := NewOutboxWorker(svc)
	ctx := context.Background()

	campID := uuid.New()
	payload, err := coldpath.MarshalOutbox(FraudThreatPayload{
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
