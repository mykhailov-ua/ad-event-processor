package management

import (
	"context"
	"testing"

	"espx/internal/database"
	"espx/internal/ingestion"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func BenchmarkGlobalSpendReconciler_ApplyBatch(b *testing.B) {
	pool, cleanupDB := database.SetupTestDB(b)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(b)
	defer cleanupRedis()

	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	const budgetLimit = int64(500_000_000)
	_, err := pool.Exec(ctx,
		"INSERT INTO customers (id, name, balance) VALUES ($1, $2, $3)",
		ingestion.ToUUID(customerID), "Bench Customer", budgetLimit)
	if err != nil {
		b.Fatal(err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		ingestion.ToUUID(campaignID), "Bench Campaign", "ACTIVE", ingestion.ToUUID(customerID), budgetLimit)
	if err != nil {
		b.Fatal(err)
	}
	if err := rdb.Set(ctx, ingestion.BudgetCampaignKey(campaignID), budgetLimit, 0).Err(); err != nil {
		b.Fatal(err)
	}

	reconciler := NewGlobalSpendReconciler(
		pool,
		[]redis.UniversalClient{rdb},
		ingestion.NewStaticSlotSharder(1),
		GlobalSpendReconcilerConfig{MinBatchSize: 100, MaxConcurrency: 8},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		txns := buildSpendSyncTxns(campaignID, 100, 1_000, "bench")
		batchKey := uuid.New().String()
		b.StartTimer()
		if err := reconciler.ApplyBatch(ctx, batchKey, txns); err != nil {
			b.Fatal(err)
		}
	}
}
