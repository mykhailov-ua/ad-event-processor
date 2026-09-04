package campaign

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type exportBatchCampaignStub struct {
	bundles map[uuid.UUID]CampaignExportBundle
}

func (s *exportBatchCampaignStub) GetCampaign(context.Context, uuid.UUID) (CampaignDTO, error) {
	return CampaignDTO{}, ErrCampaignNotFound
}

func (s *exportBatchCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (CampaignMarginDTO, error) {
	return CampaignMarginDTO{}, nil
}

func (s *exportBatchCampaignStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *exportBatchCampaignStub) ListCampaignsFiltered(context.Context, ListCampaignsFilter) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *exportBatchCampaignStub) CountCampaignStatusTotals(context.Context, ListCampaignsFilter, string, string) (CampaignStatusTotalsDTO, error) {
	return CampaignStatusTotalsDTO{}, nil
}

func (s *exportBatchCampaignStub) AttachCampaignListMarginBreach(context.Context, []CampaignDTO) {}

func (s *exportBatchCampaignStub) PatchCampaign(context.Context, uuid.UUID, PatchCampaignRequest) (CampaignDTO, error) {
	return CampaignDTO{}, nil
}

func (s *exportBatchCampaignStub) PublishCampaign(context.Context, uuid.UUID, bool) (CampaignDTO, error) {
	return CampaignDTO{}, nil
}

func (s *exportBatchCampaignStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (CampaignPublishCheckDTO, error) {
	return CampaignPublishCheckDTO{}, nil
}

func (s *exportBatchCampaignStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *exportBatchCampaignStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s *exportBatchCampaignStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *exportBatchCampaignStub) CloneCampaign(context.Context, CloneCampaignSpec) (CloneCampaignResult, error) {
	return CloneCampaignResult{}, nil
}

func (s *exportBatchCampaignStub) ExportCampaign(_ context.Context, id uuid.UUID) (CampaignExportBundle, error) {
	if bundle, ok := s.bundles[id]; ok {
		return bundle, nil
	}
	return CampaignExportBundle{}, ErrCampaignNotFound
}

func (s *exportBatchCampaignStub) ImportCampaign(context.Context, ImportCampaignSpec) (ImportCampaignResult, error) {
	return ImportCampaignResult{}, nil
}

func (s *exportBatchCampaignStub) ImportMigrationCampaigns(context.Context, ImportMigrationSpec) (ImportMigrationResult, error) {
	return ImportMigrationResult{}, nil
}

func (s *exportBatchCampaignStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (IntegrationHealthDTO, error) {
	return IntegrationHealthDTO{}, nil
}

func (s *exportBatchCampaignStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *exportBatchCampaignStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *exportBatchCampaignStub) ArchiveCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func TestExportCampaignsBatch_returnsBundlesAndPerIdErrors(t *testing.T) {
	t.Parallel()
	okID := uuid.New()
	missingID := uuid.New()
	stub := &exportBatchCampaignStub{
		bundles: map[uuid.UUID]CampaignExportBundle{
			okID: {
				ExportVersion: CampaignExportVersion,
				Campaign:      CampaignExportCampaign{Name: "Alpha"},
			},
		},
	}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/export?ids="+okID.String()+","+missingID.String(), http.NoBody)
	rec := httptest.NewRecorder()
	h.exportCampaignsBatch(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp CampaignExportBatchResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Contains(t, resp.Items, okID.String())
	assert.Equal(t, "Alpha", resp.Items[okID.String()].Campaign.Name)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, missingID.String(), resp.Errors[0].ID)
	assert.Equal(t, "not_found", resp.Errors[0].ErrorCode)
}
