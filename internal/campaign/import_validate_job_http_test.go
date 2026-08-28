package campaign

import (
	"context"
	"encoding/json"
	"fmt"
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

func writeCampaignImportValidationJSONForTest(ctx context.Context, path string, spec reportjob.ReportJobSpec) error {
	kind := migrationsource.SourceKind(strings.TrimSpace(spec.ImportSourceKind))
	if kind == "" {
		return fmt.Errorf("import_source_kind required")
	}
	payload := []byte(strings.TrimSpace(string(spec.ImportPayload)))
	if len(payload) == 0 {
		return fmt.Errorf("import_payload required")
	}
	if len(payload) > migrationsource.MaxPayloadBytes {
		return fmt.Errorf("import_payload too large")
	}
	result, err := migrationsource.Preview(kind, payload, nil)
	if err != nil {
		return errValidation(err.Error())
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o640)
}

func TestPostCampaignImportValidateJob_createsJob(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	runner := reportjob.NewReportJobRunner(t.TempDir(), reportjob.ExportDeps{
		WriteCampaignImportValidation: writeCampaignImportValidationJSONForTest,
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
