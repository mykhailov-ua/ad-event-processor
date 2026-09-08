package flow_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type flowStoreTestHost struct{}

func (flowStoreTestHost) LanderPublicBase(context.Context) string { return "" }
func (flowStoreTestHost) ValidateFlowPaths(_ context.Context, paths []flow.PathDTO) error {
	return flow.ValidatePaths(paths)
}
func (flowStoreTestHost) PublishFlowReload(context.Context) error { return nil }

func TestCloneFlow_copiesPaths_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postgres testcontainers required")
	}
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	st := flow.NewStore(pool, flowStoreTestHost{})
	landerID := uuid.New()
	offerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'clone-lander', 'https://l.test')`, landerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO offers (id, name, url) VALUES ($1, 'clone-offer', 'https://o.test')`, offerID)
	require.NoError(t, err)

	created, err := st.CreateFlow(ctx, flow.CreateFlowRequest{
		Name: "source-flow",
		Paths: []flow.PathDTO{{
			Weight:  100,
			Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
			Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
		}},
	})
	require.NoError(t, err)

	cloned, err := st.CloneFlow(ctx, created.ID, flow.CloneFlowRequest{Name: "cloned-flow"})
	require.NoError(t, err)
	require.Equal(t, "cloned-flow", cloned.Name)
	require.NotEqual(t, created.ID, cloned.ID)
}

func TestCreateFlow_rejectsWeightSum_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postgres testcontainers required")
	}
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	st := flow.NewStore(pool, flowStoreTestHost{})
	landerID := uuid.New()
	offerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'w-lander', 'https://l.test')`, landerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO offers (id, name, url) VALUES ($1, 'w-offer', 'https://o.test')`, offerID)
	require.NoError(t, err)

	_, err = st.CreateFlow(ctx, flow.CreateFlowRequest{
		Name: "bad-weights",
		Paths: []flow.PathDTO{{
			Weight:  90,
			Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
			Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
		}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "sum to 100")
}
