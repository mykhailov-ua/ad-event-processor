package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPublisher_campaignsForbiddenWithoutCampaignPerm(t *testing.T) {
	h := &campaign.CampaignsHTTPHandlers{
		Campaigns: &campaignListStub{},
		RequireAnyPermission: func(required []string, next http.HandlerFunc) http.HandlerFunc {
			publisherPerms := map[string]bool{"supply:read:scoped": true, "customers:read": true}
			return func(w http.ResponseWriter, r *http.Request) {
				allowed := false
				for _, p := range required {
					if publisherPerms[p] {
						allowed = true
						break
					}
				}
				if !allowed {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
					return
				}
				next(w, r)
			}
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

type campaignListStub struct{}

func (s campaignListStub) GetCampaign(context.Context, uuid.UUID) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s campaignListStub) GetCampaignMargin(context.Context, uuid.UUID) (campaign.CampaignMarginDTO, error) {
	return campaign.CampaignMarginDTO{}, nil
}

func (s campaignListStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s campaignListStub) ListCampaignsFiltered(context.Context, campaign.ListCampaignsFilter) ([]campaign.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s campaignListStub) CountCampaignStatusTotals(context.Context, campaign.ListCampaignsFilter, string, string) (campaign.CampaignStatusTotalsDTO, error) {
	return campaign.CampaignStatusTotalsDTO{}, nil
}

func (s campaignListStub) AttachCampaignListMarginBreach(context.Context, []campaign.CampaignDTO) {
}

func (s campaignListStub) PatchCampaign(context.Context, uuid.UUID, campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s campaignListStub) PublishCampaign(context.Context, uuid.UUID, bool) (campaign.CampaignDTO, error) {
	return campaign.CampaignDTO{}, nil
}

func (s campaignListStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	return campaign.CampaignPublishCheckDTO{Valid: true}, nil
}

func (s campaignListStub) RunCampaignSmoke(context.Context, uuid.UUID) (campaign.CampaignSmokeResultDTO, error) {
	return campaign.CampaignSmokeResultDTO{Passed: true}, nil
}

func (s campaignListStub) CreateCampaignWizardSession(context.Context, uuid.UUID, string) (campaign.CampaignWizardSessionDTO, error) {
	return campaign.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) GetCampaignWizardSession(context.Context, uuid.UUID) (campaign.CampaignWizardSessionDTO, error) {
	return campaign.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) UpdateCampaignWizardSessionStep(context.Context, uuid.UUID, string, json.RawMessage) (campaign.CampaignWizardSessionDTO, error) {
	return campaign.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) CommitCampaignWizardSession(context.Context, uuid.UUID, string, bool) (campaign.CampaignWizardCommitResult, error) {
	return campaign.CampaignWizardCommitResult{}, nil
}

func (s campaignListStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s campaignListStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]campaign.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s campaignListStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s campaignListStub) CloneCampaign(context.Context, campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaignResult{}, nil
}

func (s campaignListStub) ExportCampaign(context.Context, uuid.UUID) (campaign.CampaignExportBundle, error) {
	return campaign.CampaignExportBundle{}, nil
}

func (s campaignListStub) ImportCampaign(context.Context, campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	return campaign.ImportCampaignResult{}, nil
}

func (s campaignListStub) ImportMigrationCampaigns(context.Context, campaign.ImportMigrationSpec) (campaign.ImportMigrationResult, error) {
	return campaign.ImportMigrationResult{}, nil
}

func (s campaignListStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	return campaign.IntegrationHealthDTO{}, nil
}

func (s campaignListStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s campaignListStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s campaignListStub) ArchiveCampaign(context.Context, uuid.UUID, string) error {
	return nil
}
