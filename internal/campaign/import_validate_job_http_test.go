package campaign_test

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

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/importexport"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type campaignReaderStub struct{}

func (campaignReaderStub) GetCampaign(context.Context, uuid.UUID) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (campaignReaderStub) GetCampaignMargin(context.Context, uuid.UUID) (campaign.CampaignMarginDTO, error) {
	return campaign.CampaignMarginDTO{}, nil
}

func (campaignReaderStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (campaignReaderStub) ListCampaignsFiltered(context.Context, campaign.ListCampaignsFilter) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (campaignReaderStub) CountCampaignStatusTotals(context.Context, campaign.ListCampaignsFilter, string, string) (campaign.CampaignStatusTotalsDTO, error) {
	return campaign.CampaignStatusTotalsDTO{}, nil
}

func (campaignReaderStub) AttachCampaignListMarginBreach(context.Context, []campaign.CampaignDTO) {}

func (campaignReaderStub) PatchCampaign(context.Context, uuid.UUID, campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (campaignReaderStub) PublishCampaign(context.Context, uuid.UUID, bool) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (campaignReaderStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	return campaign.CampaignPublishCheckDTO{}, nil
}

func (campaignReaderStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (campaignReaderStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]campaign.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (campaignReaderStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (campaignReaderStub) CloneCampaign(context.Context, campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaignResult{}, nil
}

func (campaignReaderStub) ExportCampaign(context.Context, uuid.UUID) (campaign.CampaignExportBundle, error) {
	return campaign.CampaignExportBundle{}, nil
}

func (campaignReaderStub) ImportCampaign(context.Context, campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	return campaign.ImportCampaignResult{}, nil
}

func (campaignReaderStub) ImportMigrationCampaigns(context.Context, campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationResult{}, nil
}

func (campaignReaderStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	return campaign.IntegrationHealthDTO{}, nil
}

func (campaignReaderStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (campaignReaderStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func TestPostCampaignImportValidateJob_createsJob(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)

	runner := reportjob.NewReportJobRunner(t.TempDir(), reportjob.ExportDeps{
		WriteCampaignImportValidation: importexport.WriteCampaignImportValidationJSON,
	})
	h := &campaign.CampaignsHTTPHandlers{
		Campaigns:  campaignReaderStub{},
		ReportJobs: runner,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, err := json.Marshal(campaign.ImportValidateJobRequest{
		CustomerID: uuid.New().String(),
		SourceKind: string(migrationsource.SourceKindKeitaroJSON),
		Payload:    fixture,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/import/validate/jobs", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
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
