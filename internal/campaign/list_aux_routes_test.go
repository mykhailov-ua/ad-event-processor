package campaign

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type listAuxRouteCampaignStub struct{}

func (listAuxRouteCampaignStub) GetCampaign(context.Context, uuid.UUID) (CampaignDTO, error) {
	panic("list aux routes must not call GetCampaign")
}

func (listAuxRouteCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (CampaignMarginDTO, error) {
	return CampaignMarginDTO{}, nil
}

func (listAuxRouteCampaignStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (listAuxRouteCampaignStub) ListCampaignsFiltered(context.Context, ListCampaignsFilter) ([]CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (listAuxRouteCampaignStub) CountCampaignStatusTotals(context.Context, ListCampaignsFilter, string, string) (CampaignStatusTotalsDTO, error) {
	return CampaignStatusTotalsDTO{}, nil
}

func (listAuxRouteCampaignStub) AttachCampaignListMarginBreach(context.Context, []CampaignDTO) {}

func (listAuxRouteCampaignStub) PatchCampaign(context.Context, uuid.UUID, PatchCampaignRequest) (CampaignDTO, error) {
	return CampaignDTO{}, nil
}

func (listAuxRouteCampaignStub) PublishCampaign(context.Context, uuid.UUID, bool) (CampaignDTO, error) {
	return CampaignDTO{}, nil
}

func (listAuxRouteCampaignStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (CampaignPublishCheckDTO, error) {
	return CampaignPublishCheckDTO{}, nil
}

func (listAuxRouteCampaignStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (listAuxRouteCampaignStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (listAuxRouteCampaignStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (listAuxRouteCampaignStub) CloneCampaign(context.Context, CloneCampaignSpec) (CloneCampaignResult, error) {
	return CloneCampaignResult{}, nil
}

func (listAuxRouteCampaignStub) ExportCampaign(context.Context, uuid.UUID) (CampaignExportBundle, error) {
	return CampaignExportBundle{}, nil
}

func (listAuxRouteCampaignStub) ImportCampaign(context.Context, ImportCampaignSpec) (ImportCampaignResult, error) {
	return ImportCampaignResult{}, nil
}

func (listAuxRouteCampaignStub) ImportMigrationCampaigns(context.Context, ImportMigrationSpec) (ImportMigrationResult, error) {
	return ImportMigrationResult{}, nil
}

func (listAuxRouteCampaignStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (IntegrationHealthDTO, error) {
	return IntegrationHealthDTO{}, nil
}

func (listAuxRouteCampaignStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (listAuxRouteCampaignStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (listAuxRouteCampaignStub) ArchiveCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func TestListAuxRoutes_notCapturedByCampaignID_holdout(t *testing.T) {
	t.Parallel()

	h := &CampaignsHTTPHandlers{Campaigns: listAuxRouteCampaignStub{}}
	mux := http.NewServeMux()
	h.Register(mux)

	from := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)

	cases := []struct {
		name string
		path string
	}{
		{name: "list-facets", path: "/api/v1/campaigns/list-facets"},
		{name: "metrics-totals", path: "/api/v1/campaigns/metrics-totals?from=" + from + "&to=" + to},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.NotEqual(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.NotContains(t, rec.Body.String(), "invalid campaign id")
		})
	}
}
