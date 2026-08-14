package controlplane

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	ingestdb "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPgFencingCustomer(t *testing.T, pool ingestdb.DBTX, customerID uuid.UUID) {
	t.Helper()
	q := ingestdb.New(pool)
	_, err := q.CreateCustomer(context.Background(), ingestdb.CreateCustomerParams{
		ID:       domain.ToUUID(customerID),
		Name:     "pg-fencing",
		Balance:  0,
		Currency: "USD",
	})
	require.NoError(t, err)
}

func wireStalePgFencingEpoch(t *testing.T, svc *Service, rdb redis.UniversalClient) {
	t.Helper()
	ctx := context.Background()
	gate := pgfailover.NewFencingGate(rdb)
	gate.AdvanceFloor(5)
	svc.pgFencing = gate
	require.NoError(t, rdb.Set(ctx, "ad_event_processor:pg:global:fencing_epoch", "5", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ad_event_processor:pg:global:dsn_epoch", "3", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ad_event_processor:pg:global:dsn", "postgres://stale", 0).Err())
}

func TestTopUpBalance_rejectsStalePgFencingEpoch(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	seedPgFencingCustomer(t, pool, customerID)

	svc := NewService(ctx, pool, []redis.UniversalClient{rdb}, nil, &config.Config{})
	defer svc.Close()
	wireStalePgFencingEpoch(t, svc, rdb)

	err := svc.TopUpBalance(ctx, customerID, 1_000_000, "iso07-topup")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrStalePgFencingEpoch)

	var balance int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, domain.ToUUID(customerID)).Scan(&balance))
	assert.Zero(t, balance)
}

func TestApplyPaymentCredit_rejectsStalePgFencingEpoch(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	intentID := uuid.New()
	seedPgFencingCustomer(t, pool, customerID)

	svc := NewService(ctx, pool, []redis.UniversalClient{rdb}, nil, &config.Config{})
	defer svc.Close()
	wireStalePgFencingEpoch(t, svc, rdb)

	applied, ledgerID, err := svc.ApplyPaymentCredit(
		ctx,
		customerID,
		2_500_000,
		"payment:"+intentID.String(),
		intentID,
		"stripe",
		"pi_iso07",
	)
	require.Error(t, err, "harness=pg_failover_fencing: stale epoch must reject payment credit")
	require.ErrorIs(t, err, ErrStalePgFencingEpoch)
	assert.False(t, applied)
	assert.Zero(t, ledgerID)

	var balance int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT balance FROM customers WHERE id = $1`, domain.ToUUID(customerID)).Scan(&balance))
	assert.Zero(t, balance)

	var ledgerCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_ledger WHERE customer_id = $1`, domain.ToUUID(customerID)).Scan(&ledgerCount))
	assert.Zero(t, ledgerCount)
}
