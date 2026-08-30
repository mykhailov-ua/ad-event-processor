//go:build integration

package commandpalette_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/commandpalette"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPalette_search_crossCustomer_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: command palette cross-customer search isolation")
	}
	pool := setupCommandPaletteDB(t)
	ctx := context.Background()

	customerA := uuid.New()
	customerB := uuid.New()
	campaignA := uuid.New()
	campaignB := uuid.New()

	seedCustomer(t, pool, customerA, "customer-a")
	seedCustomer(t, pool, customerB, "customer-b")
	seedCampaign(t, pool, campaignA, customerA, "Camp Alpha Shared")
	seedCampaign(t, pool, campaignB, customerB, "Camp Beta Shared")

	store := commandpalette.NewStore(pool)
	items, err := store.SearchEntities(ctx, customerA, "camp", 25, []string{"campaign"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, campaignA.String(), items[0].ID)
	assert.Equal(t, "Camp Alpha Shared", items[0].Label)

	for _, item := range items {
		assert.NotEqual(t, campaignB.String(), item.ID)
	}
}

func setupCommandPaletteDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	t.Cleanup(cleanup)
	return pool
}

func seedCustomer(t *testing.T, pool *pgxpool.Pool, customerID uuid.UUID, name string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')`,
		customerID, name)
	require.NoError(t, err)
}

func seedCampaign(t *testing.T, pool *pgxpool.Pool, campaignID, customerID uuid.UUID, name string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, $2, 'ACTIVE', $3)`,
		campaignID, name, customerID)
	require.NoError(t, err)
}
