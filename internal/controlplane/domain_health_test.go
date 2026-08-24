package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestIsTLSAllowed_roles(t *testing.T) {
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	svc := &Service{pool: pool}

	_, err := pool.Exec(ctx, `
		INSERT INTO domain_health_status (hostname, role) VALUES
		('trk.example.com', 'tracking'),
		('buyer.example.com', 'custom'),
		('admin.example.com', 'admin')`)
	require.NoError(t, err)

	allowed, err := svc.IsTLSAllowed(ctx, "trk.example.com")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = svc.IsTLSAllowed(ctx, "buyer.example.com")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = svc.IsTLSAllowed(ctx, "admin.example.com")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = svc.IsTLSAllowed(ctx, "evil.example.com")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = svc.IsTLSAllowed(ctx, "")
	require.NoError(t, err)
	require.False(t, allowed)
}
