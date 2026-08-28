package governance

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCheckMediaBuyerBudgetCap_blocksIncrease(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	buyerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1,'c',100000000,'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, owner_user_id)
		VALUES ($1, 'A', 'ACTIVE', $2, 2000000, 0, $3)`, campaignID, customerID, buyerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO team_member_limits (user_id, customer_id, spend_cap_micro)
		VALUES ($1, $2, 5000000)`, buyerID, customerID)
	require.NoError(t, err)

	err = CheckMediaBuyerBudgetCap(ctx, pool, buyerID, campaignID, 6_000_000)
	require.ErrorIs(t, err, ErrBudgetApprovalRequired)

	err = CheckMediaBuyerBudgetCap(ctx, pool, buyerID, campaignID, 4_000_000)
	require.NoError(t, err)
}
