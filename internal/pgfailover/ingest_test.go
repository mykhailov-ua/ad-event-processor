package pgfailover_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/pgfailover"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestStartIngestSubscribers_reconnectOnPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	standbyDSN := pool.Config().ConnString() + "&application_name=standby"

	var calls atomic.Int32
	rt := pgfailover.StartIngestSubscribers(ctx, []redis.UniversalClient{redisClient}, pgfailover.IngestSubscriberConfig{
		MaxConns: 4,
		MinConns: 1,
		Interval: 50 * time.Millisecond,
	}, func(newPool *pgxpool.Pool) {
		calls.Add(1)
		if newPool != nil {
			newPool.Close()
		}
	})
	require.NotNil(t, rt)
	defer rt.Stop()

	require.NoError(t, pgfailover.PublishDSN(ctx, redisClient, standbyDSN, 2))

	deadline := time.Now().Add(5 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	require.GreaterOrEqual(t, int(calls.Load()), 1, "harness=pg_failover_ingest_subscriber: publish must trigger reconnect")
	require.Equal(t, standbyDSN, rt.CurrentDSN())
}
