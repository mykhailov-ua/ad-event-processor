package integration_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/stream"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ConversionLedger_twoPostbacksSameClickID(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	pool, cleanup := testutil.SetupAdsPostgres(t)
	defer cleanup()

	customerID := uuid.New()
	campID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone)
		VALUES ($1, 'ledger-test', 1000000, 'ACTIVE', $2, 'ASAP', 0, 'UTC')`, campID, customerID)
	require.NoError(t, err)

	applier := stream.NewConversionLedgerApplier(db.New(pool))
	first := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-ledger-integration",
		Payload:    []byte(`{"status":"hold","revenue_micro":"1000000"}`),
	}
	second := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-ledger-integration",
		Payload:    []byte(`{"status":"approved","revenue_micro":"2000000"}`),
	}
	applier.ApplyBatch(ctx, []*domain.Event{first, second})
	require.Contains(t, string(second.Payload), `"conversion_payout_micro":"3000000"`)
}
