package management

import (
	"espx/pkg/faultproof"

	"context"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"
	ingestdb "espx/internal/ingestion/sqlc"
	"espx/internal/management/pb"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestFault_BatchSettlementDrain(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{SettlementInternalToken: "settlement-test-token"}
	svc := NewService(pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()

	handler := NewSettlementHandler(svc, cfg)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-token", "settlement-test-token"))

	customerID := uuid.New()
	intentA := uuid.New()
	intentB := uuid.New()
	q := ingestdb.New(pool)
	_, err := q.CreateCustomer(context.Background(), ingestdb.CreateCustomerParams{
		ID:       ingestion.ToUUID(customerID),
		Name:     "batch settlement customer",
		Balance:  0,
		Currency: "EUR",
	})
	require.NoError(t, err)

	resp, err := handler.BatchApplySettlement(ctx, &pb.BatchApplySettlementRequest{
		Credits: []*pb.ApplyPaymentCreditRequest{
			{
				CustomerId:           customerID.String(),
				AmountMicro:          5_000_000,
				LedgerIdempotencyKey: "batch:credit:a",
				PaymentIntentId:      intentA.String(),
				Provider:             "stripe",
				ProviderRef:          "pi_batch_a",
			},
			{
				CustomerId:           customerID.String(),
				AmountMicro:          3_000_000,
				LedgerIdempotencyKey: "batch:credit:b",
				PaymentIntentId:      intentB.String(),
				Provider:             "stripe",
				ProviderRef:          "pi_batch_b",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.CreditResults, 2)
	require.True(t, resp.CreditResults[0].Applied)
	require.True(t, resp.CreditResults[1].Applied)
	require.Empty(t, resp.CreditResults[0].Error)
	require.Empty(t, resp.CreditResults[1].Error)

	faultproof.Log(t, "batch_settlement_drain", map[string]string{
		"subsystem": "settlement",
		"items":     "2",
		"applied":   "2",
	})
}

func TestFault_SlotMigrationCutoverInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()
	rdbs := buildFourRedisShards(rdb, nil)
	svc, _, ctx := setupSlotMigrationFault(t, rdbs)

	const slot int16 = 2
	campID, _ := seedCampaignForSlot(t, svc, svc.GetPool(), ctx, slot, rdbs[2])
	mapRepo := ingestion.NewSlotMapRepo(svc.GetPool())
	v := prepareMigratingVersion(t, ctx, mapRepo, slot, 0)
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, v))

	require.NoError(t, svc.VerifySlotMigrationR5(ctx))
	ingestion.AssertBudgetInvariant(t, ctx, svc.GetPool(), rdbs[0], campID)

	faultproof.Log(t, "slot_migration_cutover_invariant", map[string]string{
		"subsystem":   "slot_migration",
		"r5_ok":       "true",
		"campaign_id": campID.String(),
	})
}
