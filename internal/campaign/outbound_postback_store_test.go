package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReplaceCampaignOutboundPostbacks_roundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign outbound postbacks")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	key := []byte("postback-encryption-secret-key32")
	customerID := uuid.New()
	campID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone)
		VALUES ($1, 'outbound-test', 1000000, 'ACTIVE', $2, 'ASAP', 0, 'UTC')`, campID, customerID)
	require.NoError(t, err)

	saved, err := ReplaceCampaignOutboundPostbacks(ctx, pool, key, campID, []OutboundPostbackWriteDTO{
		{
			Name:         "affiliate",
			Priority:     0,
			Enabled:      true,
			Provider:     "webhook",
			URLTemplate:  "https://partner.example/postback?click={click_id}",
			TargetEvent:  "conversion",
			TriggerKind:  OutboundTriggerGoal,
			TriggerValue: "approved",
		},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.True(t, saved[0].HasAPIToken == false)

	got, err := ListCampaignOutboundPostbacks(ctx, pool, campID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "goal", got[0].TriggerKind)
	require.Equal(t, "approved", got[0].TriggerValue)
}
