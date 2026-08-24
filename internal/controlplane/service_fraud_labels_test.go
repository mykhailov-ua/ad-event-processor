package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMLManualLabels_customerScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(context.Background(), pool, nil, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	customerA := uuid.New()
	customerB := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES
			($1, 'labels-a', 0, 'USD'),
			($2, 'labels-b', 0, 'USD')`,
		domain.ToUUID(customerA), domain.ToUUID(customerB))
	require.NoError(t, err)

	ipA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ipB := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	require.NoError(t, svc.UpsertMLManualLabelForCustomer(ctx, customerA, ipA, 1, "bot"))
	require.NoError(t, svc.UpsertMLManualLabelForCustomer(ctx, customerB, ipB, 0, "ok"))

	labelsA, err := svc.ListMLManualLabelsForCustomer(ctx, customerA, 10)
	require.NoError(t, err)
	require.Len(t, labelsA, 1)
	require.Equal(t, ipA, labelsA[0].IPHash)

	labelsB, err := svc.ListMLManualLabelsForCustomer(ctx, customerB, 10)
	require.NoError(t, err)
	require.Len(t, labelsB, 1)
	require.Equal(t, ipB, labelsB[0].IPHash)

	require.NoError(t, svc.UpsertMLManualLabelForCustomer(ctx, customerA, ipA, 0, "false positive"))
	labelsA, err = svc.ListMLManualLabelsForCustomer(ctx, customerA, 10)
	require.NoError(t, err)
	require.Equal(t, 0, labelsA[0].Label)
}

func TestBulkUpsertMLManualLabelsForCustomer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	svc := NewService(context.Background(), pool, nil, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'bulk-labels', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)

	rows := []MLManualLabelInput{
		{IPHash: "cccccccccccccccccccccccccccccccc", Label: 1, Reason: "a"},
		{IPHash: "dddddddddddddddddddddddddddddddd", Label: 0, Reason: "b"},
	}
	count, err := svc.BulkUpsertMLManualLabelsForCustomer(ctx, customerID, rows)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	labels, err := svc.ListMLManualLabelsForCustomer(ctx, customerID, 10)
	require.NoError(t, err)
	require.Len(t, labels, 2)
}
