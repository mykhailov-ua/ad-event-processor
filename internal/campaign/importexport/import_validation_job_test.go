package importexport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteCampaignImportValidationJSON_invalidPayloadFails(t *testing.T) {
	t.Parallel()
	err := WriteCampaignImportValidationJSON(context.Background(), t.TempDir()+"/out.json", reportjob.ReportJobSpec{
		ReportKey:        reportjob.CampaignImportValidationReportKey,
		ImportSourceKind: "unknown-source",
		ImportPayload:    []byte(`{}`),
	})
	require.Error(t, err)
}

func TestWriteCampaignImportValidationJSON_keitaroFixture(t *testing.T) {
	t.Parallel()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixture, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	dir := t.TempDir()
	out := filepath.Join(dir, "validation.json")
	require.NoError(t, WriteCampaignImportValidationJSON(context.Background(), out, reportjob.ReportJobSpec{
		ReportKey:        reportjob.CampaignImportValidationReportKey,
		ImportSourceKind: string(migrationsource.SourceKindKeitaroJSON),
		ImportPayload:    fixture,
	}))

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	var result migrationsource.PreviewResult
	require.NoError(t, json.Unmarshal(raw, &result))
	require.NotEmpty(t, result.MappedCampaigns)
}

func TestCampaignImportValidateJob_asyncCompletes_holdout(t *testing.T) {
	t.Parallel()
	fixture, err := os.ReadFile(filepath.Join("..", "..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	runner := reportjob.NewReportJobRunner(t.TempDir(), reportjob.ExportDeps{
		WriteCampaignImportValidation: WriteCampaignImportValidationJSON,
	})
	customerID := uuid.New()
	jobID, err := runner.CreateJob(context.Background(), reportjob.ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        reportjob.CampaignImportValidationReportKey,
		Format:           "json",
		ImportSourceKind: string(migrationsource.SourceKindKeitaroJSON),
		ImportPayload:    fixture,
	}, "validate-job-idem-1")
	require.NoError(t, err)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, ok := runner.GetJob(context.Background(), jobID)
		require.True(t, ok)
		switch status.Status {
		case reportjob.JobStatusCompleted:
			require.Equal(t, reportjob.CampaignImportValidationReportKey, status.ReportKey)
			require.Equal(t, "json", status.Format)
			f, gotStatus, err := runner.OpenDownload(context.Background(), jobID)
			require.NoError(t, err)
			require.Equal(t, reportjob.JobStatusCompleted, gotStatus.Status)
			_ = f.Close()
			return
		case reportjob.JobStatusFailed:
			t.Fatalf("validation job failed: %s", status.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("validation job did not complete")
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

	runner := reportjob.NewReportJobRunner(t.TempDir(), reportjob.ExportDeps{
		Pool:                          pool,
		WriteCampaignImportValidation: WriteCampaignImportValidationJSON,
	})
	customerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'validate', 0, 'USD')`, customerID)
	require.NoError(t, err)

	jobID, err := runner.CreateJob(ctx, reportjob.ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        reportjob.CampaignImportValidationReportKey,
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
	require.Equal(t, reportjob.JobStatusFailed, status.Status)

	var after int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaigns`).Scan(&after))
	require.Equal(t, before, after)
}
