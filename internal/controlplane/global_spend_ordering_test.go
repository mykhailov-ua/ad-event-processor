package controlplane

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type evalFailOnceGlobalSpendRedis struct {
	*mockRedisForRecon
	failed bool
}

func (m *evalFailOnceGlobalSpendRedis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	if !m.failed {
		m.failed = true
		cmd := redis.NewCmd(ctx)
		cmd.SetErr(errors.New("injected global spend eval failure"))
		return cmd
	}
	if len(keys) < 2 {
		return m.mockRedisForRecon.Eval(ctx, script, keys, args...)
	}
	budgetKey := keys[0]
	markerKey := keys[1]
	amount := args[0].(int64)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[markerKey]; ok {
		cmd := redis.NewCmd(ctx, int64(0))
		return cmd
	}
	current := m.data[budgetKey]
	newVal := current - amount
	if newVal <= 0 {
		delete(m.data, budgetKey)
		m.data[markerKey] = 1
		return redis.NewCmd(ctx, int64(0))
	}
	m.data[budgetKey] = newVal
	m.data[markerKey] = 1
	return redis.NewCmd(ctx, newVal)
}

func seedGlobalSpendCampaignPG(t *testing.T, ctx context.Context, pool *pgxpool.Pool, customerID, campaignID uuid.UUID, budgetLimit int64) {
	t.Helper()
	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		domain.ToUUID(customerID), "iso03", 200_000_000)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		domain.ToUUID(campaignID), "iso03", "ACTIVE", domain.ToUUID(customerID), budgetLimit)
	require.NoError(t, err)
}

func seedGlobalSpendCampaign(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rdb redis.UniversalClient, customerID, campaignID uuid.UUID, budgetLimit int64) {
	t.Helper()
	seedGlobalSpendCampaignPG(t, ctx, pool, customerID, campaignID, budgetLimit)
	require.NoError(t, rdb.Set(ctx, domain.BudgetCampaignKey(campaignID), budgetLimit, 0).Err())
}

func makeGlobalSpendTxns(campaignID uuid.UUID, count int) []dedupkey.SpendSyncTxn {
	txns := make([]dedupkey.SpendSyncTxn, count)
	for i := range txns {
		txns[i] = dedupkey.SpendSyncTxn{
			CampaignID:  campaignID,
			AmountMicro: 10_000,
			TxnID:       "iso03-txn-" + strconv.Itoa(i),
		}
	}
	return txns
}

func TestGlobalSpendApplyBatch_pgFailurePreservesRedisBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(50_000_000)
	seedGlobalSpendCampaign(t, ctx, pool, rdb, customerID, campaignID, budgetLimit)

	reconciler := NewGlobalSpendReconciler(pool, []redis.UniversalClient{rdb}, domain.NewStaticSlotSharder(1), GlobalSpendReconcilerConfig{
		MinBatchSize:   100,
		MaxConcurrency: 4,
	})
	pool.Close()

	txns := makeGlobalSpendTxns(campaignID, 100)
	err := reconciler.ApplyBatch(ctx, "iso03-pg-fail", txns)
	require.Error(t, err)

	remaining, err := rdb.Get(ctx, domain.BudgetCampaignKey(campaignID)).Int64()
	require.NoError(t, err)
	assert.Equal(t, budgetLimit, remaining, "harness=global_spend_pg_first: PG failure must not debit Redis budget key")
}

func TestGlobalSpendApplyBatch_redisRetryAfterPgCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(50_000_000)
	const batchKey = "iso03-redis-retry"

	mockInner := newMockRedisForRecon()
	mockInner.data[domain.BudgetCampaignKey(campaignID)] = budgetLimit
	failRdb := &evalFailOnceGlobalSpendRedis{mockRedisForRecon: mockInner}

	seedGlobalSpendCampaignPG(t, ctx, pool, customerID, campaignID, budgetLimit)

	reconciler := NewGlobalSpendReconciler(pool, []redis.UniversalClient{failRdb}, domain.NewStaticSlotSharder(1), GlobalSpendReconcilerConfig{
		MinBatchSize:   100,
		MaxConcurrency: 4,
	})

	txns := makeGlobalSpendTxns(campaignID, 100)
	const totalSpend = int64(100 * 10_000)

	err := reconciler.ApplyBatch(ctx, batchKey, txns)
	require.Error(t, err)

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`,
		domain.ToUUID(campaignID)).Scan(&ledgerCount))
	assert.Equal(t, 100, ledgerCount, "PG batch must commit before Redis")

	assert.Equal(t, budgetLimit, failRdb.getVal(domain.BudgetCampaignKey(campaignID)), "Redis debit must not apply on first failure")

	require.NoError(t, reconciler.ApplyBatch(ctx, batchKey, txns))
	assert.Equal(t, budgetLimit-totalSpend, failRdb.getVal(domain.BudgetCampaignKey(campaignID)))

	domain.AssertBudgetInvariant(t, ctx, pool, failRdb, campaignID)
}
