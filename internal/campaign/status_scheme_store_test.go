package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReplaceCampaignStatusSchemeRules_roundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign status scheme rules")
	}
	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	campID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone)
		VALUES ($1, 'status-scheme-test', 1000000, 'ACTIVE', $2, 'ASAP', 0, 'UTC')`, campID, customerID)
	require.NoError(t, err)

	saved, err := ReplaceCampaignStatusSchemeRules(ctx, pool, campID, []StatusSchemeRuleDTO{
		{
			WhenStatus:        "hold",
			SetInternalStatus: "rejected",
			PayoutMode:        StatusSchemePayoutZero,
			FireOutbound:      false,
			Enabled:           true,
		},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.NotEmpty(t, saved[0].ID)

	got, err := ListCampaignStatusSchemeRules(ctx, pool, campID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "hold", got[0].WhenStatus)
	require.Equal(t, "rejected", got[0].SetInternalStatus)
	require.False(t, got[0].FireOutbound)

	ruleID, err := uuid.Parse(saved[0].ID)
	require.NoError(t, err)
	patched, err := PatchCampaignStatusSchemeRule(ctx, pool, campID, ruleID, PatchStatusSchemeRuleRequest{
		FireOutbound: boolPtr(true),
	})
	require.NoError(t, err)
	require.True(t, patched.FireOutbound)
}

func boolPtr(v bool) *bool {
	return &v
}
