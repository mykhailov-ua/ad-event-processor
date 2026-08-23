package controlplane

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMLBlacklistBatch_noNestedUpdateBlacklistOutbox(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: ml blacklist batch (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, &config.Config{})
	worker := NewOutboxWorker(svc)

	campaignID := uuid.New()
	for i := range 5 {
		require.NoError(t, svc.EnqueueFraudThreat(ctx, "blacklist", fmtIP(i), campaignID.String(), 95, 0, 3600))
	}

	var nestedBefore int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_BLACKLIST'`).Scan(&nestedBefore))

	processed, err := worker.ProcessOutboxWithCount(ctx, 20)
	require.NoError(t, err)
	require.Equal(t, 5, processed)

	var nestedAfter int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_BLACKLIST'`).Scan(&nestedAfter))
	require.Equal(t, nestedBefore, nestedAfter, "ML path must not enqueue nested UPDATE_BLACKLIST rows")

	for i := range 5 {
		ip := fmtIP(i)
		var exists bool
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM ip_blacklist WHERE ip = $1)`, ip).Scan(&exists))
		require.True(t, exists, "ip %s", ip)
		isMember, err := rdb.SIsMember(ctx, "blacklist:fraud", ip).Result()
		require.NoError(t, err)
		require.True(t, isMember, "redis ip %s", ip)
	}
}

func TestMLBlacklistBatch_replayIdempotentRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: ml blacklist replay idempotency (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, &config.Config{})
	worker := NewOutboxWorker(svc)

	ip := "203.0.113.44"
	payload, err := coldpath.MarshalOutbox(FraudThreatPayload{
		Action:     "blacklist",
		IP:         ip,
		CampaignID: uuid.New().String(),
		Score:      99,
		TTLSeconds: 600,
	})
	require.NoError(t, err)

	var eventID int64
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		VALUES ('ML_BLACKLIST_ADD', $1, 'PENDING')
		RETURNING id`, payload).Scan(&eventID))

	_, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)

	card1, err := rdb.SCard(ctx, "blacklist:fraud").Result()
	require.NoError(t, err)
	require.Equal(t, 1, int(card1))

	_, err = pool.Exec(ctx, `
		UPDATE outbox_events SET status = 'PENDING', processing_started_at = NULL WHERE id = $1`, eventID)
	require.NoError(t, err)

	_, err = worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)

	card2, err := rdb.SCard(ctx, "blacklist:fraud").Result()
	require.NoError(t, err)
	require.Equal(t, card1, card2, "replay must not grow blacklist:fraud set")
}

func TestFault_MLBlacklistBatchPropagation(t *testing.T) {
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

	svc := newBareService(t, pool, []redis.UniversalClient{rdb1, rdb2, rdb3}, &config.Config{})
	worker := NewOutboxWorker(svc)
	ctx := context.Background()

	campaignID := uuid.New()
	for i := range 10 {
		require.NoError(t, svc.EnqueueFraudThreat(ctx, "blacklist", fmtIP(i+20), campaignID.String(), 95, 0, 1800))
	}

	start := time.Now()
	processed, err := worker.ProcessOutboxWithCount(ctx, 20)
	require.NoError(t, err)
	elapsed := time.Since(start)
	require.Equal(t, 10, processed)

	for i := range 10 {
		ip := fmtIP(i + 20)
		for j, rdb := range []redis.UniversalClient{rdb1, rdb2, rdb3} {
			ok, getErr := rdb.SIsMember(ctx, "blacklist:fraud", ip).Result()
			require.NoError(t, getErr, "shard %d", j)
			require.True(t, ok, "shard %d ip %s", j, ip)
		}
	}

	require.Less(t, elapsed, 30*time.Second)

	faultproof.Log(t, "ml_blacklist_batch", map[string]string{
		"subsystem":   "management",
		"shards":      "3",
		"ips":         "10",
		"elapsed_ms":  elapsed.String(),
		"sla_seconds": "30",
	})
}

func fmtIP(n int) string {
	return fmt.Sprintf("203.0.113.%d", n%256)
}
