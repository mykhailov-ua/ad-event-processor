package management

import (
	"espx/pkg/faultproof"

	"context"
	"strconv"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_GlobalSpendReconciler(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(50_000_000)
	_, err = pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		ingestion.ToUUID(customerID), "MR Spend", 200_000_000)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		ingestion.ToUUID(campaignID), "MR Campaign", "ACTIVE", ingestion.ToUUID(customerID), budgetLimit)
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, ingestion.BudgetCampaignKey(campaignID), budgetLimit, 0).Err())

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0, GlobalSpendBatchMin: 100}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	reconciler := NewGlobalSpendReconciler(pool, []redis.UniversalClient{rdb}, ingestion.NewStaticSlotSharder(1), GlobalSpendReconcilerConfig{
		MinBatchSize:   100,
		MaxConcurrency: 8,
	})
	svc.SetGlobalSpendReconciler(reconciler)

	txns := make([]dedupkey.SpendSyncTxn, 100)
	for i := range txns {
		txns[i] = dedupkey.SpendSyncTxn{
			CampaignID:  campaignID,
			AmountMicro: 10_000,
			TxnID:       "mr-txn-" + strconv.Itoa(i),
		}
	}
	payload, err := dedupkey.EncodeSpendSyncPayload(txns)
	require.NoError(t, err)

	var buf [4096 + 64]byte
	factorU := dedupkey.FactorU(dedupkey.WriteCanonicalProxyBatchPayload(buf[:0], 99, payload))
	in := RegionIngestBatchInput{
		RegionCode: 1,
		NodeID:     "proxy-spend-1",
		Seq:        99,
		FactorU:    factorU,
		Payload:    payload,
	}

	for i := 0; i < 3; i++ {
		_, err = svc.IngestRegionProxyBatch(ctx, in)
		require.NoError(t, err)
	}

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM balance_ledger WHERE campaign_id = $1 AND type = 'FEE'`, ingestion.ToUUID(campaignID),
	).Scan(&ledgerCount))
	require.Equal(t, 100, ledgerCount)

	ingestion.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)

	faultproof.Log(t, "mr_global_spend_reconciler", map[string]string{
		"subsystem":   "global_spend_reconciler",
		"ledger_rows": strconv.Itoa(ledgerCount),
		"batch_txns":  "100",
		"baseline_ok": "true",
		"budget_ok":   "true",
	})
}
