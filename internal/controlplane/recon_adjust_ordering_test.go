package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type evalFailOnceRedis struct {
	*mockRedisForRecon
	failed bool
}

func (m *evalFailOnceRedis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	if !m.failed {
		m.failed = true
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(errors.New("injected redis eval failure"))
		return cmd
	}
	return m.mockRedisForRecon.Eval(ctx, script, keys, args...)
}

func (m *evalFailOnceRedis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
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

func (m *evalFailOnceRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
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

func TestApplyReconciliationAdjust_pgFailurePreservesRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	campID := uuid.New()
	customerID := uuid.New()
	const beforeSync = int64(2_500_000)

	payload, err := coldpath.MarshalOutbox(ReconciliationAdjustPayload{
		CampaignID: campID.String(),
		CustomerID: customerID.String(),
		ShardID:    0,
		LedgerAmt:  -500_000,
		RedisDelta: 500_000,
		Reason:     "iso01_pg_failure_harness",
	})
	require.NoError(t, err)

	require.NoError(t, rdb.Set(ctx, domain.CampaignSyncKey(campID), beforeSync, 0).Err())

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, nil)
	ob := NewOutboxWorker(svc)

	err = ob.ApplyReconciliationAdjust(ctx, 42, payload)
	require.Error(t, err)

	after, err := rdb.Get(ctx, domain.CampaignSyncKey(campID)).Int64()
	require.NoError(t, err)
	assert.Equal(t, beforeSync, after, "harness=pg_outbox_recon: PG failure must not move Redis sync key")

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM balance_ledger WHERE idempotency_hash = $1`,
		reconciliationAdjustIdempotencyHash(42),
	).Scan(&ledgerCount))
	assert.Equal(t, 0, ledgerCount)
}

func TestApplyReconciliationAdjust_redisRetryAfterPgCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = 10_000_000
	const beforeSync = int64(1_000_000)

	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'iso01-retry', 100000000, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window, updated_at)
		VALUES ($1, 'iso01-retry', $2, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400, NOW())`,
		domain.ToUUID(campaignID), budgetLimit, domain.ToUUID(customerID))
	require.NoError(t, err)

	mockInner := newMockRedisForRecon()
	mockInner.data[domain.CampaignSyncKey(campaignID)] = beforeSync

	failRdb := &evalFailOnceRedis{mockRedisForRecon: mockInner}
	svc := newBareService(t, pool, []redis.UniversalClient{failRdb}, nil)
	ob := NewOutboxWorker(svc)

	payload, err := coldpath.MarshalOutbox(ReconciliationAdjustPayload{
		CampaignID: campaignID.String(),
		CustomerID: customerID.String(),
		ShardID:    0,
		LedgerAmt:  -500_000,
		RedisDelta: 500_000,
		Reason:     "iso01_redis_retry_harness",
	})
	require.NoError(t, err)

	err = ob.ApplyReconciliationAdjust(ctx, 77, payload)
	require.Error(t, err)

	var spend int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`,
		domain.ToUUID(campaignID)).Scan(&spend))
	assert.Equal(t, int64(500_000), spend, "PG phase must commit before Redis")

	mid := failRdb.getVal(domain.CampaignSyncKey(campaignID))
	assert.Equal(t, beforeSync, mid, "Redis adjust must not apply on first failure")

	require.NoError(t, ob.ApplyReconciliationAdjust(ctx, 77, payload))

	after := failRdb.getVal(domain.CampaignSyncKey(campaignID))
	assert.Equal(t, beforeSync+500_000, after)

	domain.AssertBudgetInvariant(t, ctx, pool, failRdb, campaignID)
}
