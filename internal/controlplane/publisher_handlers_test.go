package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPublisher_campaignsForbiddenWithoutCampaignPerm(t *testing.T) {
	h := &controlplane.CampaignsHTTPHandlers{
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

func (s campaignListStub) GetCampaign(context.Context, uuid.UUID) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (s campaignListStub) GetCampaignMargin(context.Context, uuid.UUID) (controlplane.CampaignMarginDTO, error) {
	return controlplane.CampaignMarginDTO{}, nil
}

func (s campaignListStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]controlplane.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (s campaignListStub) AttachCampaignListMarginBreach(context.Context, []controlplane.CampaignDTO) {}

func (s campaignListStub) PatchCampaign(context.Context, uuid.UUID, controlplane.PatchCampaignRequest) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (s campaignListStub) PublishCampaign(context.Context, uuid.UUID, bool) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (s campaignListStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (controlplane.CampaignPublishCheckDTO, error) {
	return controlplane.CampaignPublishCheckDTO{Valid: true}, nil
}

func (s campaignListStub) RunCampaignSmoke(context.Context, uuid.UUID) (controlplane.CampaignSmokeResultDTO, error) {
	return controlplane.CampaignSmokeResultDTO{Passed: true}, nil
}

func (s campaignListStub) CreateCampaignWizardSession(context.Context, uuid.UUID, string) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) GetCampaignWizardSession(context.Context, uuid.UUID) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) UpdateCampaignWizardSessionStep(context.Context, uuid.UUID, string, json.RawMessage) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (s campaignListStub) CommitCampaignWizardSession(context.Context, uuid.UUID, string, bool) (controlplane.CampaignWizardCommitResult, error) {
	return controlplane.CampaignWizardCommitResult{}, nil
}

func (s campaignListStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s campaignListStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]controlplane.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (s campaignListStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (s campaignListStub) CloneCampaign(context.Context, controlplane.CloneCampaignSpec) (controlplane.CloneCampaignResult, error) {
	return controlplane.CloneCampaignResult{}, nil
}

func (s campaignListStub) ExportCampaign(context.Context, uuid.UUID) (controlplane.CampaignExportBundle, error) {
	return controlplane.CampaignExportBundle{}, nil
}

func (s campaignListStub) ImportCampaign(context.Context, controlplane.ImportCampaignSpec) (controlplane.ImportCampaignResult, error) {
	return controlplane.ImportCampaignResult{}, nil
}

func (s campaignListStub) ImportMigrationCampaigns(context.Context, controlplane.ImportMigrationSpec) (controlplane.ImportMigrationResult, error) {
	return controlplane.ImportMigrationResult{}, nil
}

func (s campaignListStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (controlplane.IntegrationHealthDTO, error) {
	return controlplane.IntegrationHealthDTO{}, nil
}

func (s campaignListStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (s campaignListStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}
