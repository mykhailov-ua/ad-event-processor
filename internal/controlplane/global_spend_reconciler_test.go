package controlplane

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGlobalSpendTest(t *testing.T) (context.Context, *pgxpool.Pool, redis.UniversalClient, uuid.UUID) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	t.Cleanup(cleanupDB)
	rdb, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(100_000_000)
	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		domain.ToUUID(customerID), "Global Spend Customer", 500_000_000)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		domain.ToUUID(campaignID), "Global Spend Campaign", "ACTIVE", domain.ToUUID(customerID), budgetLimit)
	require.NoError(t, err)

	require.NoError(t, rdb.Set(ctx, domain.BudgetCampaignKey(campaignID), budgetLimit, 0).Err())
	return ctx, pool, rdb, campaignID
}

func buildSpendSyncTxns(campaignID uuid.UUID, count int, amountMicro int64, prefix string) []dedupkey.SpendSyncTxn {
	txns := make([]dedupkey.SpendSyncTxn, count)
	for i := range txns {
		txns[i] = dedupkey.SpendSyncTxn{
			CampaignID:  campaignID,
			AmountMicro: amountMicro,
			TxnID:       fmt.Sprintf("%s-txn-%d", prefix, i),
		}
	}
	return txns
}

func TestGlobalSpendReconciler_ApplyBatch(t *testing.T) {
	ctx, pool, rdb, campaignID := setupGlobalSpendTest(t)

	const (
		txnCount   = 100
		deltaMicro = int64(1_000)
	)
	totalMicro := int64(txnCount) * deltaMicro

	reconciler := NewGlobalSpendReconciler(
		pool,
		[]redis.UniversalClient{rdb},
		domain.NewStaticSlotSharder(1),
		GlobalSpendReconcilerConfig{MinBatchSize: 100, MaxConcurrency: 4},
	)

	txns := buildSpendSyncTxns(campaignID, txnCount, deltaMicro, "batch1")
	start := time.Now()
	require.NoError(t, reconciler.ApplyBatch(ctx, "batch-1", txns))
	elapsed := time.Since(start)
	assert.Less(t, elapsed, time.Second, "cross-region apply must complete within 1000 ms SLA")

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, domain.ToUUID(campaignID),
	).Scan(&ledgerCount))
	assert.Equal(t, txnCount, ledgerCount)

	var currentSpend int64
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT current_spend FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID),
	).Scan(&currentSpend))
	assert.Equal(t, totalMicro, currentSpend)

	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)

	require.NoError(t, reconciler.ApplyBatch(ctx, "batch-1", txns))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, domain.ToUUID(campaignID),
	).Scan(&ledgerCount))
	assert.Equal(t, txnCount, ledgerCount)
}

func TestGlobalSpendReconciler_ConcurrentApply(t *testing.T) {
	ctx, pool, rdb, campaignID := setupGlobalSpendTest(t)

	const (
		workers    = 24
		txnCount   = 100
		deltaMicro = int64(500)
	)

	reconciler := NewGlobalSpendReconciler(
		pool,
		[]redis.UniversalClient{rdb},
		domain.NewStaticSlotSharder(1),
		GlobalSpendReconcilerConfig{MinBatchSize: 100, MaxConcurrency: 8},
	)

	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			txns := buildSpendSyncTxns(campaignID, txnCount, deltaMicro, fmt.Sprintf("w%d", workerID))
			if err := reconciler.ApplyBatch(ctx, fmt.Sprintf("batch-%d", workerID), txns); err != nil {
				errCh <- err
			}
		}(w)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	expectedRows := workers * txnCount
	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, domain.ToUUID(campaignID),
	).Scan(&ledgerCount))
	assert.Equal(t, expectedRows, ledgerCount)

	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)
}
