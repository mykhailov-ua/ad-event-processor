package controlplane

import (
	"context"
	"testing"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/domain"
	ingestdb "espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettlementHandler_GetLedgerEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{SettlementInternalToken: "settlement-test-token"}
	svc := NewService(pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()

	handler := NewSettlementHandler(svc, cfg)
	settlement := handler.PaymentSettlement()
	ctx := context.Background()

	missingIntentID := uuid.New()
	missingEntry, err := settlement.GetLedgerEntry(ctx, missingIntentID)
	require.NoError(t, err)
	assert.False(t, missingEntry.Found)

	customerID := uuid.New()
	intentID := uuid.New()
	q := ingestdb.New(pool)
	_, err = q.CreateCustomer(context.Background(), ingestdb.CreateCustomerParams{
		ID:       domain.ToUUID(customerID),
		Name:     "ledger lookup customer",
		Balance:  0,
		Currency: "USD",
	})
	require.NoError(t, err)

	applied, ledgerID, err := svc.ApplyPaymentCredit(
		context.Background(),
		customerID,
		12_500_000,
		"payment:"+intentID.String(),
		intentID,
		"stripe",
		"pi_test_ledger_lookup",
	)
	require.NoError(t, err)
	require.True(t, applied)
	require.NotZero(t, ledgerID)

	foundEntry, err := settlement.GetLedgerEntry(ctx, intentID)
	require.NoError(t, err)
	require.True(t, foundEntry.Found)
	require.True(t, foundEntry.HasTopup)
	assert.Equal(t, int64(12_500_000), foundEntry.TopupAmountMicro)
	assert.Zero(t, foundEntry.RefundTotalMicro)
}
