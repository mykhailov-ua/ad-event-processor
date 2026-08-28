package reconciliation

import (
	"context"
	"strconv"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_GlobalSpendReconciler(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(50_000_000)
	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		domain.ToUUID(customerID), "MR Spend", 200_000_000)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		domain.ToUUID(campaignID), "MR Campaign", "ACTIVE", domain.ToUUID(customerID), budgetLimit)
	require.NoError(t, err)
	require.NoError(t, redisClient.Set(ctx, domain.BudgetCampaignKey(campaignID), budgetLimit, 0).Err())

	reconciler := NewGlobalSpendReconciler(pool, []redis.UniversalClient{redisClient}, domain.NewStaticSlotSharder(1), GlobalSpendReconcilerConfig{
		MinBatchSize:   100,
		MaxConcurrency: 8,
	})

	txns := make([]dedupkey.SpendSyncTxn, 100)
	for i := range txns {
		txns[i] = dedupkey.SpendSyncTxn{
			CampaignID:  campaignID,
			AmountMicro: 10_000,
			TxnID:       "mr-txn-" + strconv.Itoa(i),
		}
	}

	for n := range 3 {
		require.NoError(t, reconciler.ApplyBatch(ctx, "mr-batch-"+strconv.Itoa(n), txns))
	}

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, domain.ToUUID(campaignID),
	).Scan(&ledgerCount))
	require.Equal(t, 100, ledgerCount)

	domain.AssertBudgetInvariant(t, ctx, pool, redisClient, campaignID)

	faultproof.Log(t, "mr_global_spend_reconciler", map[string]string{
		"subsystem":   "global_spend_reconciler",
		"ledger_rows": strconv.Itoa(ledgerCount),
		"batch_txns":  "100",
		"baseline_ok": "true",
		"budget_ok":   "true",
	})
}
