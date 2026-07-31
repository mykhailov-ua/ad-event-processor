package controlplane

import (
	"context"
	"testing"

	"espx/internal/config"
	"espx/internal/domain"
	"espx/internal/ledger"
	"espx/internal/testutil"
	"espx/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_MarginGuardPause(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := testutil.SetupAdsPostgres(t)
	defer cleanupDB()
	rdb, cleanupRedis := testutil.SetupRedis(t)
	defer cleanupRedis()

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

	var policyID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, 'ledger-guard', 500, true)
		RETURNING id`, campaignID).Scan(&policyID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, created_at)
		VALUES ($1, $2, -100000, 'FEE', 'mg-fee-1', now()),
		       ($1, $2, 120000, 'rtb_cost', 'mg-rtb-1', now()),
		       ($1, $2, 120000, 'publisher_payout', 'mg-pub-1', now())
	`, customerID, campaignID)
	require.NoError(t, err)

	cfg := &config.Config{MarginGuardDefaultThresholdBps: 500}
	worker := ledger.NewWorker(pool, nil, cfg, nil, nil)
	require.NoError(t, worker.RunCycle(ctx))

	var outboxCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events
		WHERE event_type = 'PAUSE_CAMPAIGN' AND payload::text LIKE $1`,
		"%"+campaignID.String()+"%",
	).Scan(&outboxCount))
	require.Equal(t, 1, outboxCount)

	domain.AssertBudgetInvariant(t, ctx, pool, rdb, campaignID)

	faultproof.Log(t, "margin_guard_pause", map[string]string{
		"campaign_id": campaignID.String(),
		"policy_id":   policyID.String(),
	})
}
