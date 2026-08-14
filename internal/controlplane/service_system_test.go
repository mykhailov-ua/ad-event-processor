package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockIPUsesOutbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, domain.NewJumpHashSharder(1), nil)
	defer svc.Close()

	ctx := context.Background()
	require.NoError(t, svc.BlockIP(ctx, "10.0.0.1", "fraud"))

	var outboxCount int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events
		WHERE event_type = 'UPDATE_BLACKLIST' AND status IN ('PENDING', 'PROCESSING')`).Scan(&outboxCount)
	require.NoError(t, err)
	assert.Equal(t, 1, outboxCount)

	assert.Eventually(t, func() bool {
		isMember, err := rdb.SIsMember(ctx, "blacklist:fraud", "10.0.0.1").Result()
		return err == nil && isMember
	}, 2*time.Second, 20*time.Millisecond)
}

func TestBlockIP_ProtectedAndAudit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, domain.NewJumpHashSharder(1), nil)
	defer svc.Close()

	ctx := context.Background()

	err := svc.BlockIP(ctx, "8.8.8.8", "fraud")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "protected by allowlist")

	var blacklistCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ip_blacklist WHERE ip = '8.8.8.8'").Scan(&blacklistCount)
	require.NoError(t, err)
	assert.Equal(t, 0, blacklistCount)

	ip := "198.51.100.5"
	err = svc.BlockIPWithTTL(ctx, ip, "fraud", nil)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ip_blacklist WHERE ip = $1", ip).Scan(&blacklistCount)
	require.NoError(t, err)
	assert.Equal(t, 1, blacklistCount)

	var auditCount int
	var reasonID, source string
	err = pool.QueryRow(ctx, "SELECT COUNT(*), reason_id, source FROM edge_block_audit WHERE ip = $1 GROUP BY reason_id, source", ip).Scan(&auditCount, &reasonID, &source)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
	assert.Equal(t, "fraud", reasonID)
	assert.Equal(t, "fraud", source)
}
