package domains

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBurnDomain_blocksTLS_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postgres testcontainers required")
	}
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	dh := NewDomainHealth(poolBanHost{pool: pool})
	poolID := uuid.MustParse("55555555-5555-4555-8555-555555555555")
	host := "burn-me.example"
	_, err := pool.Exec(ctx, `
		INSERT INTO domain_pools (id, name) VALUES ($1, 'burn-test-pool')
		ON CONFLICT (name) DO NOTHING`, poolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_pool_domains (pool_id, hostname, sort_order, status)
		VALUES ($1, $2, 0, 'active')
		ON CONFLICT (hostname) DO UPDATE SET status = 'active'`, poolID, host)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_health_status (hostname, role) VALUES ($1, 'custom')`, host)
	require.NoError(t, err)

	allowed, err := dh.IsTLSAllowed(ctx, host)
	require.NoError(t, err)
	require.True(t, allowed)

	resp, err := dh.BurnDomain(ctx, host, BurnDomainRequest{})
	require.NoError(t, err)
	require.Equal(t, "banned", resp.PoolStatus)

	allowed, err = dh.IsTLSAllowed(ctx, host)
	require.NoError(t, err)
	require.False(t, allowed)
}
