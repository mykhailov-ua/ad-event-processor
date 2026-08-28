package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandleMediaBuyerBudgetIncrease_autoDenyExceedsCap_holdout(t *testing.T) {
	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	buyerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency, allowed_overdraft) VALUES ($1,'c',100000000,'USD',0)`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, owner_user_id)
		VALUES ($1, 'A', 'ACTIVE', $2, 2000000, 0, $3)`, campaignID, customerID, buyerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO team_member_limits (user_id, customer_id, spend_cap_micro)
		VALUES ($1, $2, 5000000)`, buyerID, customerID)
	require.NoError(t, err)

	svc := &Service{pool: pool}
	locked, err := svc.GetCampaignRow(ctx, campaignID)
	require.NoError(t, err)
	err = svc.handleMediaBuyerBudgetIncrease(ctx, locked, buyerID, 6_000_000)
	require.ErrorIs(t, err, ErrBudgetApprovalAutoDenied)

	var status string
	err = pool.QueryRow(ctx, `
		SELECT status FROM team_budget_approvals
		WHERE campaign_id = $1 ORDER BY created_at DESC LIMIT 1`, campaignID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "DENIED", status)
}

func TestWriteCampaignImportValidationJSON_invalidPayloadFails(t *testing.T) {
	t.Parallel()
	err := writeCampaignImportValidationJSON(context.Background(), t.TempDir()+"/out.json", ReportJobSpec{
		ReportKey:        campaignImportValidationReportKey,
		ImportSourceKind: "unknown-source",
		ImportPayload:    []byte(`{}`),
	})
	require.Error(t, err)
}
