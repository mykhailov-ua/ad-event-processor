package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestImportMigrationCampaigns_invalidPayloadNoCampaignImport_holdout(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.ImportMigrationCampaigns(t.Context(), ImportMigrationSpec{
		CustomerID:     uuid.New(),
		IdempotencyKey: "batch-1",
		SourceKind:     migrationsource.SourceKind("unknown-source"),
		Payload:        []byte(`{}`),
	})
	require.Error(t, err)
}

func TestPreviewMigrationPull_invalidPayloadNoPG_holdout(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.PreviewMigrationPull(t.Context(), PullMigrationPreviewSpec{
		SourceKind: migrationsource.SourceKind("unknown-source"),
		BaseURL:    "https://example.test",
	})
	require.Error(t, err)
}

func TestCampaignImportValidateJob_failedValidationNoCampaignRows_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: import validation job holdout leaves campaigns unchanged")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	var before int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaigns`).Scan(&before))

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		Pool:                          pool,
		WriteCampaignImportValidation: writeCampaignImportValidationJSON,
	})
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'validate', 0, 'USD')`, customerID)
	require.NoError(t, err)

	jobID, err := runner.CreateJob(ctx, ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        campaignImportValidationReportKey,
		Format:           "json",
		ImportSourceKind: "unknown-source",
		ImportPayload:    []byte(`{}`),
	}, "")
	require.NoError(t, err)

	n, err := runner.ProcessOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	status, ok := runner.GetJob(ctx, jobID)
	require.True(t, ok)
	require.Equal(t, JobStatusFailed, status.Status)

	var after int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaigns`).Scan(&after))
	require.Equal(t, before, after)
}
