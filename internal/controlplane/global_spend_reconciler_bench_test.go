package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

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
		domain.ToUUID(customerID), "Bench Customer", budgetLimit)
	if err != nil {
		b.Fatal(err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO campaigns (id, name, status, customer_id, budget_limit) VALUES ($1, $2, $3, $4, $5)",
		domain.ToUUID(campaignID), "Bench Campaign", "ACTIVE", domain.ToUUID(customerID), budgetLimit)
	if err != nil {
		b.Fatal(err)
	}
	if err := rdb.Set(ctx, domain.BudgetCampaignKey(campaignID), budgetLimit, 0).Err(); err != nil {
		b.Fatal(err)
	}

	reconciler := NewGlobalSpendReconciler(
		pool,
		[]redis.UniversalClient{rdb},
		domain.NewStaticSlotSharder(1),
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
