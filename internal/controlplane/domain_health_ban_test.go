package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"
	"ad-event-processor/pkg/domainhealth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyReputationToProbe(t *testing.T) {
	t.Parallel()
	res := domainhealth.Result{HealthStatus: domainhealth.HealthHealthy, ProbeDetail: "http 200"}
	applyReputationToProbe(&res, true, "safe_browsing:MALWARE")
	require.Equal(t, domainhealth.HealthDown, res.HealthStatus)
	require.Equal(t, "reputation:safe_browsing:MALWARE", res.ProbeDetail)
}

func TestDomainHealth_markPoolDomainBanned(t *testing.T) {
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	svc := &Service{pool: pool}
	poolID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	_, err := pool.Exec(ctx, `
		INSERT INTO domain_pools (id, name) VALUES ($1, 'ban-test-pool')
		ON CONFLICT (name) DO NOTHING`, poolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_pool_domains (pool_id, hostname, sort_order, status)
		VALUES ($1, 'ban-me.example', 0, 'active')
		ON CONFLICT (hostname) DO UPDATE SET status = 'active'`, poolID)
	require.NoError(t, err)

	require.NoError(t, svc.markPoolDomainBanned(ctx, "ban-me.example"))

	var status string
	err = pool.QueryRow(ctx, `
		SELECT status FROM domain_pool_domains WHERE hostname = 'ban-me.example'`).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "banned", status)
}

func TestDomainHealth_reputationUnsafeBansPool(t *testing.T) {
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	svc := &Service{pool: pool}
	poolID := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	_, err := pool.Exec(ctx, `
		INSERT INTO domain_pools (id, name) VALUES ($1, 'rep-ban-pool')
		ON CONFLICT (name) DO NOTHING`, poolID)
	require.NoError(t, err)
	host := "rep-ban.example"
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_pool_domains (pool_id, hostname, sort_order, status)
		VALUES ($1, $2, 0, 'active')
		ON CONFLICT (hostname) DO UPDATE SET status = 'active'`, poolID, host)
	require.NoError(t, err)

	res := domainhealth.Result{HealthStatus: domainhealth.HealthHealthy}
	applyReputationToProbe(&res, true, "safe_browsing:MALWARE")
	require.Equal(t, domainhealth.HealthDown, res.HealthStatus)
	require.NoError(t, svc.markPoolDomainBanned(ctx, host))

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM domain_pool_domains WHERE hostname = $1`, host).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "banned", status)
}
