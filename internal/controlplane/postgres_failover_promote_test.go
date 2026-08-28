package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestBuildPgFailoverPromoter_simplePathRequiresWritable(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	dsn := pool.Config().ConnString()
	svc := &Service{
		cfg: &config.Config{
			PostgresStandbyDSN:           config.Secret(dsn),
			DBTrackerMaxConns:            4,
			DBMinConns:                   1,
			PostgresPromoteCommand:       "",
			PostgresFailoverSnapshotSync: false,
		},
	}

	var reconnected bool
	promoter := buildPgFailoverPromoter(svc, func(newPool *pgxpool.Pool) {
		reconnected = true
		if newPool != nil {
			newPool.Close()
		}
	})

	gotDSN, err := promoter.Promote(ctx)
	require.NoError(t, err, "harness=pg_failover_promote: writable standby must promote")
	require.Equal(t, dsn, gotDSN)
	require.True(t, reconnected)
}
