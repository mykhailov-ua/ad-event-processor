package payment

import (
	"context"
	"testing"
	"time"

	"espx/internal/ingestion"
	ads_db "espx/internal/domain/db"
	"espx/internal/payment/db"
	"espx/internal/payment/dbtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func seedReconCustomer(t *testing.T, pool *pgxpool.Pool, customerID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := ads_db.New(pool).CreateCustomer(ctx, ads_db.CreateCustomerParams{
		ID:       ingestion.ToUUID(customerID),
		Name:     "recon test customer",
		Balance:  0,
		Currency: "USD",
	})
	require.NoError(t, err)
}

func countReconFindingsByKind(t *testing.T, pool *pgxpool.Pool, runID int64, kind db.PaymentFinancialFindingKind) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM payment.financial_recon_findings
		WHERE run_id = $1 AND kind = $2`, runID, kind).Scan(&n)
	require.NoError(t, err)
	return n
}

func TestFinancialReconRun_persistsRunAndFindings(t *testing.T) {
	if testing.Short() {
		t.Skip("requires testcontainers")
	}

	pool, cleanup := dbtest.SetupTestDB(t)
	defer cleanup()

	recon := NewReconService(pool, nil, nil)
	end := time.Now().UTC()
	summary, err := recon.Run(context.Background(), end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.NotZero(t, summary.RunID)

	var status string
	var findingsCount int
	err = pool.QueryRow(context.Background(), `
		SELECT status, findings_count FROM payment.financial_recon_runs WHERE id = $1`, summary.RunID).
		Scan(&status, &findingsCount)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", status)
	require.Equal(t, summary.FindingsCount, findingsCount)
}

func TestFinancialReconRun_missingTopupAfterWebhook(t *testing.T) {
	if testing.Short() {
		t.Skip("requires testcontainers")
	}

	pool, cleanup := dbtest.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	customerID := uuid.New()
	seedReconCustomer(t, pool, customerID)

	svc := NewService(pool, NewMockProvider(), nil)
	result, err := svc.CreatePaymentIntent(ctx, customerID, 11_000_000, "USD", "recon-unit-"+uuid.New().String(), nil)
	require.NoError(t, err)
	providerRef := result.Intent.ProviderRef.String
	payload := `{"id":"evt_recon_unit","type":"payment_intent.succeeded","data":{"object":{"id":"` + providerRef + `","amount":11000000}}}`
	err = svc.ProcessStripeWebhook(ctx, "evt_recon_unit", "payment_intent.succeeded", []byte(payload), providerRef, 11_000_000, payload)
	require.NoError(t, err)

	recon := NewReconService(pool, nil, nil)
	end := time.Now().UTC()
	summary, err := recon.Run(ctx, end.Add(-time.Hour), end)
	require.NoError(t, err)
	require.Equal(t, 1, countReconFindingsByKind(t, pool, summary.RunID, db.PaymentFinancialFindingKindMISSINGLEDGERTOPUP))
}
