package controlplane

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"ad-event-processor/internal/testutil"
)

func TestReplaceCampaignConversionMappings_roundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign conversion mappings")
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
		VALUES ($1, 'c1', 1000000, 'ACTIVE', $2, 'ASAP', 0, 'UTC')`, campID, customerID)
	require.NoError(t, err)

	svc := &Service{pool: pool}
	saved, err := svc.ReplaceCampaignConversionMappings(ctx, campID, []ConversionMappingDTO{
		{InboundStatus: "lead", GoalName: "lead", PayoutMicro: 2_500_000},
		{InboundStatus: "hold", GoalName: "hold", PayoutMicro: 0},
	})
	require.NoError(t, err)
	require.Len(t, saved, 2)

	got, err := svc.ListCampaignConversionMappings(ctx, campID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	var leadPayout int64
	for _, row := range got {
		if row.InboundStatus == "lead" {
			leadPayout = row.PayoutMicro
		}
	}
	require.Equal(t, int64(2_500_000), leadPayout)
}
