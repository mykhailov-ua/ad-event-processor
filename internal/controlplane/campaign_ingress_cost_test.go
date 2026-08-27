package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyCampaignIngressCostPatch_roundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign ingress_cost_config patch")
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
	err = svc.applyCampaignIngressCostPatch(ctx, campID, IngressCostConfigDTO{
		Param:    "cost",
		Scale:    "decimal",
		MaxMicro: 5_000_000,
		Policy:   "ignore",
	})
	require.NoError(t, err)

	got, err := svc.GetCampaign(ctx, campID)
	require.NoError(t, err)
	require.NotNil(t, got.IngressCostConfig)
	require.Equal(t, "cost", got.IngressCostConfig.Param)
	require.Equal(t, int64(5_000_000), got.IngressCostConfig.MaxMicro)

	parsed := domain.ParseIngressCostConfigJSON([]byte(`{"param":"cost","scale":"decimal","max_micro":5000000,"policy":"ignore"}`))
	require.True(t, parsed.Enabled())

	var raw []byte
	err = pool.QueryRow(ctx, `SELECT ingress_cost_config FROM campaigns WHERE id = $1`, campID).Scan(&raw)
	require.NoError(t, err)
	registryCfg := domain.ParseIngressCostConfigJSON(raw)
	require.Equal(t, parsed.Param, registryCfg.Param)
	require.Equal(t, parsed.MaxMicro, registryCfg.MaxMicro)
}
