package reportjob

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/integrations/googlesheets"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandlers_postGoogleSheetWithoutOAuth_returns400_holdout(t *testing.T) {
	t.Parallel()
	prevActor := ExportActorLabel
	ExportActorLabel = func(context.Context) string { return uuid.New().String() }
	t.Cleanup(func() { ExportActorLabel = prevActor })

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		ValidateGoogleSheetsOAuth: func(ctx context.Context, operatorUserID string) error {
			return googlesheets.ErrNotConnected
		},
	})
	h := &HTTPHandlers{Runner: runner}
	mux := http.NewServeMux()
	h.Register(mux)

	body, err := json.Marshal(ReportJobSpec{
		CustomerID:  uuid.New().String(),
		ReportKey:   "placements",
		From:        time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:          time.Now().UTC().Format(time.RFC3339),
		Format:      "csv",
		Destination: "google_sheet",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "not connected")
}

func TestHTTPHandlers_downloadGoogleSheetJob_returns409_holdout(t *testing.T) {
	prevActor := ExportActorLabel
	ExportActorLabel = func(context.Context) string { return uuid.New().String() }
	t.Cleanup(func() { ExportActorLabel = prevActor })

	runner := NewReportJobRunner(t.TempDir(), ExportDeps{
		ValidateGoogleSheetsOAuth: func(ctx context.Context, operatorUserID string) error {
			return nil
		},
		WriteGoogleSheet: func(ctx context.Context, spec ReportJobSpec) (GoogleSheetExportResult, error) {
			return GoogleSheetExportResult{
				SpreadsheetID:  "sheet-id",
				SpreadsheetURL: "https://docs.google.com/spreadsheets/d/sheet-id",
			}, nil
		},
	})
	jobID, err := runner.CreateJob(context.Background(), ReportJobSpec{
		CustomerID:  uuid.New().String(),
		ReportKey:   "placements",
		From:        time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		To:          time.Now().UTC().Format(time.RFC3339),
		Format:      "csv",
		Destination: "google_sheet",
		ExportedBy:  uuid.New().String(),
	}, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		status, ok := runner.GetJob(context.Background(), jobID)
		return ok && status.Status == JobStatusCompleted
	}, time.Second, 10*time.Millisecond)

	h := &HTTPHandlers{Runner: runner}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/jobs/"+jobID+"/download", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "USE_SPREADSHEET_URL")
}

func TestReportJobDownloadContentType_xlsx_holdout(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		reportJobDownloadContentType(ReportJobStatusDTO{Format: "xlsx"}),
	)
}
