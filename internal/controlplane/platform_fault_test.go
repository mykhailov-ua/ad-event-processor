package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment"
	ingestdb "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_BatchSettlementDrain(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{SettlementInternalToken: "settlement-test-token"}
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()

	handler := NewSettlementHandler(svc, cfg)
	ctx := context.Background()

	customerID := uuid.New()
	intentA := uuid.New()
	intentB := uuid.New()
	q := ingestdb.New(pool)
	_, err := q.CreateCustomer(context.Background(), ingestdb.CreateCustomerParams{
		ID:       domain.ToUUID(customerID),
		Name:     "batch settlement customer",
		Balance:  0,
		Currency: "EUR",
	})
	require.NoError(t, err)

	result := handler.BatchApplySettlement(ctx, payment.SettlementBatchParams{
		Credits: []payment.SettlementCreditParams{
			{
				CustomerID:           customerID,
				AmountMicro:          5_000_000,
				LedgerIdempotencyKey: "batch:credit:a",
				PaymentIntentID:      intentA,
				Provider:             "stripe",
				ProviderRef:          "pi_batch_a",
			},
			{
				CustomerID:           customerID,
				AmountMicro:          3_000_000,
				LedgerIdempotencyKey: "batch:credit:b",
				PaymentIntentID:      intentB,
				Provider:             "stripe",
				ProviderRef:          "pi_batch_b",
			},
		},
	})
	require.Len(t, result.CreditResults, 2)
	require.True(t, result.CreditResults[0].Applied)
	require.True(t, result.CreditResults[1].Applied)
	require.NoError(t, result.CreditResults[0].Err)
	require.NoError(t, result.CreditResults[1].Err)

	faultproof.Log(t, "batch_settlement_drain", map[string]string{
		"subsystem": "settlement",
		"items":     "2",
		"applied":   "2",
	})
}

func TestFault_SlotMigrationCutoverInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()
	redisShards := buildFourRedisShards(redisClient, nil)
	svc, _, ctx := setupSlotMigrationFault(t, redisShards)

	const slot int16 = 2
	campID, _ := seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, redisShards[2])
	mapRepo := domain.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 0)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))

	require.NoError(t, svc.VerifySlotMigrationR5(ctx))
	domain.AssertBudgetInvariant(t, ctx, svc.GetPool(), redisShards[0], campID)

	faultproof.Log(t, "slot_migration_cutover_invariant", map[string]string{
		"subsystem":   "slot_migration",
		"r5_ok":       "true",
		"campaign_id": campID.String(),
	})
}
