package ledger

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestFault_LedgerMarginBatchPause(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanup := setupBillingTestDB(t)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance) VALUES ($1, 'margin-guard', 1000000000)
	`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend)
		VALUES ($1, 'mg', 'ACTIVE', $2, 1000000000, 0)
	`, campaignID, customerID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, 'ledger-guard', 500, true)
	`, campaignID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, created_at)
		VALUES ($1, $2, -100000, 'FEE', 'mg-fee-1', now()),
		 ($1, $2, 120000, 'rtb_cost', 'mg-rtb-1', now()),
		 ($1, $2, 120000, 'publisher_payout', 'mg-pub-1', now())
	`, customerID, campaignID)
	require.NoError(t, err)

	windowStart := time.Now().Add(-ledgerMarginWindow).UTC()
	sumRows, err := db.New(pool).SumCampaignMarginWindowByCampaignIDs(ctx, db.SumCampaignMarginWindowByCampaignIDsParams{
		CampaignIds: []pgtype.UUID{{Bytes: campaignID, Valid: true}},
		WindowStart: pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	require.NoError(t, err)
	require.Len(t, sumRows, 1)
	require.Equal(t, int64(100_000), sumRows[0].AdvertiserSpendMicro)
	require.Equal(t, int64(120_000), sumRows[0].RtbCostMicro)

	cfg := &config.Config{MarginGuardDefaultThresholdBps: 500}
	worker := NewWorker(pool, nil, cfg, nil, nil)
	require.NoError(t, worker.RunCycle(ctx))

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events
		WHERE event_type = 'PAUSE_CAMPAIGN' AND convert_from(payload, 'UTF8') LIKE $1`,
		"%"+campaignID.String()+"%",
	).Scan(&outboxCount))
	require.Equal(t, 1, outboxCount)
}
