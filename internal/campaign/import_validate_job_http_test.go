package campaign

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostCampaignImportValidateJob_createsJob(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	runner := reportjob.NewReportJobRunner(t.TempDir(), reportjob.ExportDeps{
		WriteCampaignImportValidation: WriteCampaignImportValidationJSON,
	})
	h := &CampaignsHTTPHandlers{ReportJobs: runner}

	body, err := json.Marshal(ImportValidateJobRequest{
		CustomerID: uuid.New().String(),
		SourceKind: string(migrationsource.SourceKindKeitaroJSON),
		Payload:    fixture,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/import/validate/jobs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.postCampaignImportValidateJob(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var status reportjob.ReportJobStatusDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &status))
	require.Equal(t, reportjob.CampaignImportValidationReportKey, status.ReportKey)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, ok := runner.GetJob(context.Background(), status.ID)
		require.True(t, ok)
		switch got.Status {
		case reportjob.JobStatusCompleted, reportjob.JobStatusFailed:
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("validation job did not finish")
}
