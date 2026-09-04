package campaign

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type patchRevisionCampaignStub struct {
	campaign   CampaignDTO
	patchCalls int
}

func (s *patchRevisionCampaignStub) GetCampaign(context.Context, uuid.UUID) (CampaignDTO, error) {
	return s.campaign, nil
}

func (s *patchRevisionCampaignStub) PatchCampaign(context.Context, uuid.UUID, PatchCampaignRequest) (CampaignDTO, error) {
	s.patchCalls++
	return s.campaign, nil
}

func (s *patchRevisionCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (CampaignMarginDTO, error) {
	return CampaignMarginDTO{}, nil
}

func (s *patchRevisionCampaignStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *patchRevisionCampaignStub) ListCampaignsFiltered(context.Context, ListCampaignsFilter) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s *patchRevisionCampaignStub) CountCampaignStatusTotals(context.Context, ListCampaignsFilter, string, string) (CampaignStatusTotalsDTO, error) {
	return CampaignStatusTotalsDTO{}, nil
}

func (s *patchRevisionCampaignStub) AttachCampaignListMarginBreach(context.Context, []CampaignDTO) {}

func (s *patchRevisionCampaignStub) PublishCampaign(context.Context, uuid.UUID, bool) (CampaignDTO, error) {
	return s.campaign, nil
}

func (s *patchRevisionCampaignStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (CampaignPublishCheckDTO, error) {
	return CampaignPublishCheckDTO{Valid: true}, nil
}

func (s *patchRevisionCampaignStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *patchRevisionCampaignStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s *patchRevisionCampaignStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *patchRevisionCampaignStub) CloneCampaign(context.Context, CloneCampaignSpec) (CloneCampaignResult, error) {
	return CloneCampaignResult{}, nil
}

func (s *patchRevisionCampaignStub) ExportCampaign(context.Context, uuid.UUID) (CampaignExportBundle, error) {
	return CampaignExportBundle{}, nil
}

func (s *patchRevisionCampaignStub) ImportCampaign(context.Context, ImportCampaignSpec) (ImportCampaignResult, error) {
	return ImportCampaignResult{}, nil
}

func (s *patchRevisionCampaignStub) ImportMigrationCampaigns(context.Context, ImportMigrationSpec) (ImportMigrationResult, error) {
	return ImportMigrationResult{}, nil
}

func (s *patchRevisionCampaignStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (IntegrationHealthDTO, error) {
	return IntegrationHealthDTO{}, nil
}

func (s *patchRevisionCampaignStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *patchRevisionCampaignStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *patchRevisionCampaignStub) ArchiveCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func TestPatchCampaign_staleIfMatch_skipsPatch_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	stub := &patchRevisionCampaignStub{
		campaign: CampaignDTO{
			ID:        campID.String(),
			UpdatedAt: "2026-08-27T10:00:00Z",
			Revision:  "2026-08-27T10:00:00Z",
			Name:      "Live",
		},
	}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campID.String(), strings.NewReader(`{"name":"Draft"}`))
	req.SetPathValue("id", campID.String())
	req.Header.Set("If-Match", "stale-revision")
	rec := httptest.NewRecorder()
	h.patchCampaign(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, 0, stub.patchCalls)
	require.Contains(t, rec.Body.String(), "campaign_revision_conflict")
	require.Contains(t, rec.Body.String(), "2026-08-27T10:00:00Z")
}

func TestPatchCampaign_staleIfMatch_recordsRevisionConflict_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	stub := &patchRevisionCampaignStub{
		campaign: CampaignDTO{
			ID:        campID.String(),
			UpdatedAt: "2026-08-27T10:00:00Z",
			Revision:  "2026-08-27T10:00:00Z",
			Name:      "Live",
		},
	}
	recorded := false
	h := &CampaignsHTTPHandlers{
		Campaigns: stub,
		RecordRevisionConflict: func(_ context.Context, id uuid.UUID, expected string) {
			recorded = true
			require.Equal(t, campID, id)
			require.Equal(t, "stale-revision", expected)
		},
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campID.String(), strings.NewReader(`{"name":"Draft"}`))
	req.SetPathValue("id", campID.String())
	req.Header.Set("If-Match", "stale-revision")
	rec := httptest.NewRecorder()
	h.patchCampaign(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.True(t, recorded)
}

func TestPatchCampaign_matchingIfMatch_appliesPatch(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	revision := "2026-08-27T10:00:00Z"
	stub := &patchRevisionCampaignStub{
		campaign: CampaignDTO{
			ID:        campID.String(),
			UpdatedAt: revision,
			Revision:  revision,
			Name:      "Live",
		},
	}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campID.String(), strings.NewReader(`{"name":"Draft"}`))
	req.SetPathValue("id", campID.String())
	req.Header.Set("If-Match", revision)
	rec := httptest.NewRecorder()
	h.patchCampaign(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, stub.patchCalls)
}

func TestPatchCampaign_sameRevisionRetry_idempotent_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	revision := "2026-08-27T10:00:00Z"
	stub := &patchRevisionCampaignStub{
		campaign: CampaignDTO{
			ID:        campID.String(),
			UpdatedAt: revision,
			Revision:  revision,
			Name:      "Live",
		},
	}
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	body := `{"name":"Draft"}`
	for i := range 2 {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campID.String(), strings.NewReader(body))
		req.SetPathValue("id", campID.String())
		req.Header.Set("If-Match", revision)
		rec := httptest.NewRecorder()
		h.patchCampaign(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "attempt %d", i+1)
	}
	require.Equal(t, 2, stub.patchCalls)
}
