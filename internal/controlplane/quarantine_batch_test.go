package controlplane

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/edge"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type quarantinePublishCounter struct {
	redis.UniversalClient
	count *atomic.Int64
}

func (c *quarantinePublishCounter) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	if channel == edge.FraudQuarantineChannel {
		c.count.Add(1)
	}
	return c.UniversalClient.Publish(ctx, channel, message)
}

func TestQuarantineBatch_boundedPublishCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: quarantine batch publish budget (run make test-integration)")
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

	const n = 500
	events := make([]db.OutboxEvent, n)
	for i := range n {
		payload, err := coldpath.MarshalOutbox(BlacklistPayload{
			Action: "add",
			IP:     fmt.Sprintf("203.0.113.%d", i%256),
			Reason: "fraud",
		})
		require.NoError(t, err)
		events[i] = db.OutboxEvent{Payload: payload}
	}

	require.NoError(t, worker.applyBlacklistPayloadsBatch(ctx, events))

	const shardCount = 3
	require.Equal(t, int64(shardCount), publishCount.Load(),
		"fraud:quarantine publish must be once per shard, not per IP")

	for i := range 10 {
		ip := fmt.Sprintf("203.0.113.%d", i)
		ok, err := rdb1.SIsMember(ctx, "blacklist:fraud", ip).Result()
		require.NoError(t, err)
		require.True(t, ok, "ip %s", ip)
	}
}

func TestFault_QuarantineBatchBoundedPublish(t *testing.T) {
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

	var publishCount atomic.Int64
	wrap := func(rdb redis.UniversalClient) redis.UniversalClient {
		return &quarantinePublishCounter{UniversalClient: rdb, count: &publishCount}
	}

	ctx := context.Background()
	svc := newBareService(t, pool, []redis.UniversalClient{
		wrap(rdb1), wrap(rdb2), wrap(rdb3),
	}, &config.Config{})
	worker := NewOutboxWorker(svc)

	const n = 500
	events := make([]db.OutboxEvent, n)
	for i := range n {
		payload, err := coldpath.MarshalOutbox(BlacklistPayload{
			Action: "add",
			IP:     fmt.Sprintf("198.51.100.%d", i%256),
			Reason: "fraud",
		})
		require.NoError(t, err)
		events[i] = db.OutboxEvent{Payload: payload}
	}

	require.NoError(t, worker.applyBlacklistPayloadsBatch(ctx, events))

	const shardCount = 3
	published := publishCount.Load()
	require.Equal(t, int64(shardCount), published)

	faultproof.Log(t, "quarantine_batch_publish", map[string]string{
		"subsystem":      "management",
		"ips":            fmt.Sprintf("%d", n),
		"shards":         fmt.Sprintf("%d", shardCount),
		"publish_count":  fmt.Sprintf("%d", published),
		"publish_budget": fmt.Sprintf("%d", shardCount),
	})
}
