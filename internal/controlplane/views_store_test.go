package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestViewsStore_PG_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Views PG', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	store := reports.NewViewsStore(pool)
	created, err := store.CreateView(ctx, reports.CreateViewRequest{
		CustomerID: custID.String(),
		Name:       "Weekly placements",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{"limit":25}`),
		IsShared:   true,
	}, "owner-1", authz.MaskFull)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	list, err := store.ListView(ctx, custID.String())
	require.NoError(t, err)
	require.Len(t, list, 1)

	got, err := store.GetView(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.Name, got.Name)

	updated, err := store.UpdateView(ctx, created.ID, reports.UpdateViewRequest{
		Name:      "Updated",
		ReportKey: "placements",
		Spec:      json.RawMessage(`{"limit":50}`),
		IsShared:  false,
	}, authz.MaskFull)
	require.NoError(t, err)
	require.Equal(t, "Updated", updated.Name)

	require.NoError(t, store.DeleteView(ctx, created.ID))
	_, err = store.GetView(ctx, created.ID)
	require.ErrorIs(t, err, reports.ErrViewNotFound)
}
