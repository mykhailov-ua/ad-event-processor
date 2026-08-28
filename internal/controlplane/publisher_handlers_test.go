package controlplane_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type publisherStub struct {
	bind controlplane.PublisherBind
	err  error
}

func (s *publisherStub) ResolvePublisherBind(_ context.Context, _ uuid.UUID) (controlplane.PublisherBind, error) {
	if s.err != nil {
		return controlplane.PublisherBind{}, s.err
	}
	return s.bind, nil
}

func (s *publisherStub) GetPublisherDashboard(_ context.Context, _ controlplane.PublisherBind, _, _ time.Time) (controlplane.PublisherDashboardDTO, error) {
	return controlplane.PublisherDashboardDTO{SellerID: s.bind.SellerID}, nil
}

func (s *publisherStub) ListPublisherStatements(_ context.Context, _ controlplane.PublisherBind, _, _ time.Time, _, _ int32) ([]controlplane.PublisherStatementDTO, int64, error) {
	return nil, 0, nil
}

func TestPublisherDashboard_requiresSellerBind(t *testing.T) {
	userID := uuid.New()
	h := &controlplane.PublisherHTTPHandlers{
		Publisher: &publisherStub{err: controlplane.ErrPublisherScopeRequired},
		ActorUserID: func(_ *http.Request) (uuid.UUID, bool) {
			return userID, true
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapPublisherTestError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publisher/dashboard", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

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

func (campaignListStub) GetCampaign(context.Context, uuid.UUID) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (campaignListStub) GetCampaignMargin(context.Context, uuid.UUID) (controlplane.CampaignMarginDTO, error) {
	return controlplane.CampaignMarginDTO{}, nil
}

func (campaignListStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]controlplane.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (campaignListStub) AttachCampaignListMarginBreach(context.Context, []controlplane.CampaignDTO) {}

func (campaignListStub) PatchCampaign(context.Context, uuid.UUID, controlplane.PatchCampaignRequest) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (campaignListStub) PublishCampaign(context.Context, uuid.UUID, bool) (controlplane.CampaignDTO, error) {
	return controlplane.CampaignDTO{}, nil
}

func (campaignListStub) EvaluateCampaignPublish(context.Context, uuid.UUID) (controlplane.CampaignPublishCheckDTO, error) {
	return controlplane.CampaignPublishCheckDTO{Valid: true}, nil
}

func (campaignListStub) RunCampaignSmoke(context.Context, uuid.UUID) (controlplane.CampaignSmokeResultDTO, error) {
	return controlplane.CampaignSmokeResultDTO{Passed: true}, nil
}

func (campaignListStub) CreateCampaignWizardSession(context.Context, uuid.UUID, string) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (campaignListStub) GetCampaignWizardSession(context.Context, uuid.UUID) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (campaignListStub) UpdateCampaignWizardSessionStep(context.Context, uuid.UUID, string, json.RawMessage) (controlplane.CampaignWizardSessionDTO, error) {
	return controlplane.CampaignWizardSessionDTO{}, nil
}

func (campaignListStub) CommitCampaignWizardSession(context.Context, uuid.UUID, string, bool) (controlplane.CampaignWizardCommitResult, error) {
	return controlplane.CampaignWizardCommitResult{}, nil
}

func (campaignListStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (campaignListStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]controlplane.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (campaignListStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func (campaignListStub) CloneCampaign(context.Context, controlplane.CloneCampaignSpec) (controlplane.CloneCampaignResult, error) {
	return controlplane.CloneCampaignResult{}, nil
}

func (campaignListStub) ExportCampaign(context.Context, uuid.UUID) (controlplane.CampaignExportBundle, error) {
	return controlplane.CampaignExportBundle{}, nil
}

func (campaignListStub) ImportCampaign(context.Context, controlplane.ImportCampaignSpec) (controlplane.ImportCampaignResult, error) {
	return controlplane.ImportCampaignResult{}, nil
}

func (campaignListStub) ImportMigrationCampaigns(context.Context, controlplane.ImportMigrationSpec) (controlplane.ImportMigrationResult, error) {
	return controlplane.ImportMigrationResult{}, nil
}

func (campaignListStub) GetCampaignIntegrationHealth(context.Context, uuid.UUID) (controlplane.IntegrationHealthDTO, error) {
	return controlplane.IntegrationHealthDTO{}, nil
}

func (campaignListStub) PauseCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func (campaignListStub) ResumeCampaign(context.Context, uuid.UUID, string) error {
	return nil
}

func mapPublisherTestError(err error) (status int, code string, message string) {
	if errors.Is(err, controlplane.ErrPublisherScopeRequired) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL", "internal error"
}
