package controlplane

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type incrFailOnceQuotaRedis struct {
	*mockRedisForRecon
	failed bool
}

func (m *incrFailOnceQuotaRedis) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	if !m.failed {
		m.failed = true
		cmd := redis.NewIntCmd(ctx)
		cmd.SetErr(errors.New("injected quota incrby failure"))
		return cmd
	}
	m.mu.Lock()
	m.data[key] += value
	m.mu.Unlock()
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(m.data[key])
	return cmd
}

func (m *incrFailOnceQuotaRedis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if len(keys) == 0 {
		cmd.SetVal(0)
		return cmd
	}
	m.mu.Lock()
	_, ok := m.data[keys[0]]
	m.mu.Unlock()
	if ok {
		cmd.SetVal(1)
	} else {
		cmd.SetVal(0)
	}
	return cmd
}

func (m *incrFailOnceQuotaRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	m.mu.Lock()
	switch v := value.(type) {
	case int:
		m.data[key] = int64(v)
	case int64:
		m.data[key] = v
	case string:
		if v == "1" {
			m.data[key] = 1
		}
	}
	m.mu.Unlock()
	cmd.SetVal("OK")
	return cmd
}

func TestApplyQuotaRepair_topUpRedis_pgFailurePreservesQuotaKey(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	campID := uuid.New()
	const beforeQuota = int64(1_500_000)
	quotaKey := fmt.Sprintf("budget:quota:%s", campID.String())
	require.NoError(t, rdb.Set(ctx, quotaKey, beforeQuota, 0).Err())

	payload, err := coldpath.MarshalOutbox(QuotaRepairPayload{
		CampaignID:  campID.String(),
		ShardID:     0,
		Action:      string(QuotaRepairTopUpRedis),
		RepairMicro: 500_000,
		Reason:      "iso02_pg_failure_harness",
	})
	require.NoError(t, err)

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, nil)
	ob := NewOutboxWorker(svc)
	pool.Close()

	err = ob.ApplyQuotaRepair(ctx, 42, payload)
	require.Error(t, err)

	after, err := rdb.Get(ctx, quotaKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, beforeQuota, after, "harness=pg_outbox_quota_repair: PG Begin failure must not INCRBY quota key")
}

func TestApplyQuotaRepair_topUpRedis_retryAfterPgCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	campID := uuid.New()
	customerID := uuid.New()
	const beforeQuota = int64(2_000_000)
	quotaKey := fmt.Sprintf("budget:quota:%s", campID.String())

	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'iso02-retry', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window, updated_at)
		VALUES ($1, 'iso02-retry', 10000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400, NOW())`,
		domain.ToUUID(campID), domain.ToUUID(customerID))
	require.NoError(t, err)

	mockRdb := newMockRedisForRecon()
	mockRdb.data[quotaKey] = beforeQuota
	failRdb := &incrFailOnceQuotaRedis{mockRedisForRecon: mockRdb}

	svc := newBareService(t, pool, []redis.UniversalClient{failRdb}, nil)
	ob := NewOutboxWorker(svc)

	payload, err := coldpath.MarshalOutbox(QuotaRepairPayload{
		CampaignID:  campID.String(),
		ShardID:     0,
		Action:      string(QuotaRepairTopUpRedis),
		RepairMicro: 500_000,
		Reason:      "iso02_redis_retry_harness",
	})
	require.NoError(t, err)

	err = ob.ApplyQuotaRepair(ctx, 88, payload)
	require.Error(t, err)

	var auditCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM admin_audit_log
		WHERE action = 'QUOTA_REPAIR_TOPUP' AND target_id = $1
		  AND (metadata->>'outbox_event_id')::bigint = 88`,
		domain.ToUUID(campID)).Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "PG audit must commit before Redis")

	assert.Equal(t, beforeQuota, failRdb.getVal(quotaKey), "Redis INCRBY must not apply on first failure")

	require.NoError(t, ob.ApplyQuotaRepair(ctx, 88, payload))
	assert.Equal(t, beforeQuota+500_000, failRdb.getVal(quotaKey))
}
