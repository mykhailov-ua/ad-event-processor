package editor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type diffCampaignStub struct {
	byID map[uuid.UUID]campaign.CampaignDTO
}

func (s *diffCampaignStub) GetCampaign(_ context.Context, id uuid.UUID) (campaign.CampaignDTO, error) {
	camp, ok := s.byID[id]
	if !ok {
		return campaign.CampaignDTO{}, campaign.ErrCampaignNotFound
	}
	return camp, nil
}

func (s *diffCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (campaign.CampaignMarginDTO, error) {
	return campaign.CampaignMarginDTO{}, nil
}

func (s *diffCampaignStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *diffCampaignStub) ListCampaignsFiltered(context.Context, campaign.ListCampaignsFilter) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *diffCampaignStub) CountCampaignStatusTotals(context.Context, campaign.ListCampaignsFilter, string, string) (campaign.CampaignStatusTotalsDTO, error) {
	return campaign.CampaignStatusTotalsDTO{}, nil
}

func (s *diffCampaignStub) AttachCampaignListMarginBreach(context.Context, []campaign.CampaignDTO) {}

func (s *diffCampaignStub) PatchCampaign(context.Context, uuid.UUID, campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s *diffCampaignStub) PublishCampaign(context.Context, uuid.UUID, bool) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s *diffCampaignStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	return campaign.CampaignPublishCheckDTO{}, nil
}

func (s *diffCampaignStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *diffCampaignStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]campaign.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s *diffCampaignStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *diffCampaignStub) CloneCampaign(context.Context, campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaignResult{}, nil
}

func (s *diffCampaignStub) ExportCampaign(context.Context, uuid.UUID) (campaign.CampaignExportBundle, error) {
	return campaign.CampaignExportBundle{}, nil
}

func (s *diffCampaignStub) ImportCampaign(context.Context, campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	return campaign.ImportCampaignResult{}, nil
}

func (s *diffCampaignStub) ImportMigrationCampaigns(context.Context, campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationResult{}, nil
}

func (s *diffCampaignStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	return campaign.IntegrationHealthDTO{}, nil
}

func (s *diffCampaignStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *diffCampaignStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *diffCampaignStub) ArchiveCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func TestGetCampaignDiff_selfReturnsEmptyRows_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	stub := &diffCampaignStub{byID: map[uuid.UUID]campaign.CampaignDTO{
		campID: {ID: campID.String(), Name: "Live", CustomerID: uuid.New().String()},
	}}
	h := &campaign.CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/diff?against="+campID.String(), http.NoBody)
	req.SetPathValue("id", campID.String())
	rec := httptest.NewRecorder()
	getCampaignDiff(h, rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp CampaignDiffResponseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.Rows)
}

func TestGetCampaignDiff_crossCustomerReturns404_holdout(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	stub := &diffCampaignStub{byID: map[uuid.UUID]campaign.CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: uuid.New().String()},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: uuid.New().String()},
	}}
	h := &campaign.CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	getCampaignDiff(h, rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCampaignDiff_forbiddenAgainstReturns404_holdout(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	customerID := uuid.New().String()
	stub := &diffCampaignStub{byID: map[uuid.UUID]campaign.CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: customerID},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: customerID},
	}}
	h := &campaign.CampaignsHTTPHandlers{
		Campaigns: stub,
		AuthorizeCampaignAccess: func(_ *http.Request, id uuid.UUID) error {
			if id == rightID {
				return campaign.ErrForbidden
			}
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	getCampaignDiff(h, rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCampaignDiff_differentFieldsReturnRows(t *testing.T) {
	t.Parallel()
	leftID := uuid.New()
	rightID := uuid.New()
	customerID := uuid.New().String()
	stub := &diffCampaignStub{byID: map[uuid.UUID]campaign.CampaignDTO{
		leftID:  {ID: leftID.String(), Name: "Left", CustomerID: customerID, Status: "ACTIVE"},
		rightID: {ID: rightID.String(), Name: "Right", CustomerID: customerID, Status: "PAUSED"},
	}}
	h := &campaign.CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+leftID.String()+"/diff?against="+rightID.String(), http.NoBody)
	req.SetPathValue("id", leftID.String())
	rec := httptest.NewRecorder()
	getCampaignDiff(h, rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp CampaignDiffResponseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Rows)
}
