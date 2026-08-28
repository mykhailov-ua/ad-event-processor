package reconciliation

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRuns_Management(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	host := newTestHost(t, pool, []redis.UniversalClient{redisClient}, cfg)
	host.paymentPool = pool

	start := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO recon_runs (period_start, period_end, status, total_delta, campaigns_checked, discrepancies_found, completed_at)
		VALUES ($1, $2, 'COMPLETED', 5000, 10, 1, NOW())`, start, end)
	require.NoError(t, err)

	runs, total, err := ListRuns(context.Background(), host, "management", 50, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, runs, 1)
	assert.Equal(t, "management", runs[0].Service)
	assert.Equal(t, "COMPLETED", runs[0].Status)
}
