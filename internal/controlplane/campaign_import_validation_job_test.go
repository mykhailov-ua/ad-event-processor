package controlplane

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteCampaignImportValidationJSON_keitaroFixture(t *testing.T) {
	t.Parallel()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixture, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	dir := t.TempDir()
	out := filepath.Join(dir, "validation.json")
	require.NoError(t, writeCampaignImportValidationJSON(context.Background(), out, ReportJobSpec{
		ReportKey:        campaignImportValidationReportKey,
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
	fixture, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		WriteCampaignImportValidation: writeCampaignImportValidationJSON,
	})
	customerID := uuid.New()
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        campaignImportValidationReportKey,
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
		case JobStatusCompleted:
			require.Equal(t, campaignImportValidationReportKey, status.ReportKey)
			require.Equal(t, "json", status.Format)
			f, gotStatus, err := runner.OpenDownload(context.Background(), jobID)
			require.NoError(t, err)
			require.Equal(t, JobStatusCompleted, gotStatus.Status)
			_ = f.Close()
			return
		case JobStatusFailed:
			t.Fatalf("validation job failed: %s", status.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("validation job did not complete")
}
