package pgfailover_test

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/pgfailover"

	"github.com/stretchr/testify/require"
)

func TestEnsureWritablePrimary_acceptsWritableDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	require.NoError(t, pgfailover.EnsureWritablePrimary(ctx, pool))

	var inRecovery bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT pg_catalog.pg_is_in_recovery()`).Scan(&inRecovery))
	require.False(t, inRecovery)
}
