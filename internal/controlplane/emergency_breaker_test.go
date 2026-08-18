package controlplane

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmergencyCircuitBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		RateLimitPerMin:   100,
		RateLimitWindowMs: 60000,
	}

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()

	ctx := context.Background()

	require.NoError(t, svc.ToggleEmergencyBreaker(ctx, true, "high CPU fraud spike"))

	var dbVal string
	err := pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = 'emergency_breaker'").Scan(&dbVal)
	require.NoError(t, err)
	assert.Equal(t, "true", dbVal)

	var auditCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_log WHERE action = 'EMERGENCY_BREAKER_TOGGLED'").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), auditCount)

	var outboxCount int64
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'UPDATE_SETTINGS'").Scan(&outboxCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), outboxCount)

	worker := NewOutboxWorker(svc)
	err = worker.ProcessOutbox(ctx)
	require.NoError(t, err)

	redisVal, err := rdb.HGet(ctx, "config:values", "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, "true", redisVal)

	require.NoError(t, svc.ToggleEmergencyBreaker(ctx, false, "mitigation completed"))

	err = worker.ProcessOutbox(ctx)
	require.NoError(t, err)

	redisVal, err = rdb.HGet(ctx, "config:values", "emergency_breaker").Result()
	require.NoError(t, err)
	assert.Equal(t, "false", redisVal)
}
