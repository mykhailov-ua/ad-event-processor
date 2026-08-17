package adminapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type publisherStub struct {
	bind adminapi.PublisherBind
	err  error
}

func (s *publisherStub) ResolvePublisherBind(_ context.Context, _ uuid.UUID) (adminapi.PublisherBind, error) {
	if s.err != nil {
		return adminapi.PublisherBind{}, s.err
	}
	return s.bind, nil
}

func (s *publisherStub) GetPublisherDashboard(_ context.Context, _ adminapi.PublisherBind, _, _ time.Time) (adminapi.PublisherDashboardDTO, error) {
	return adminapi.PublisherDashboardDTO{SellerID: s.bind.SellerID}, nil
}

func (s *publisherStub) ListPublisherStatements(_ context.Context, _ adminapi.PublisherBind, _, _ time.Time, _, _ int32) ([]adminapi.PublisherStatementDTO, int64, error) {
	return nil, 0, nil
}

func TestPublisherDashboard_requiresSellerBind(t *testing.T) {
	userID := uuid.New()
	h := &adminapi.PublisherHTTPHandlers{
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
	h := &adminapi.CampaignsHTTPHandlers{
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

func (campaignListStub) GetCampaign(context.Context, uuid.UUID) (adminapi.CampaignDTO, error) {
	return adminapi.CampaignDTO{}, nil
}

func (campaignListStub) GetCampaignMargin(context.Context, uuid.UUID) (adminapi.CampaignMarginDTO, error) {
	return adminapi.CampaignMarginDTO{}, nil
}

func (campaignListStub) ListCampaigns(context.Context, uuid.UUID, string, int32, int32) ([]adminapi.CampaignDTO, int64, error) {
	return nil, 0, nil
}

func (campaignListStub) AttachCampaignListMarginBreach(context.Context, []adminapi.CampaignDTO) {}

func (campaignListStub) PatchCampaign(context.Context, uuid.UUID, adminapi.PatchCampaignRequest) (adminapi.CampaignDTO, error) {
	return adminapi.CampaignDTO{}, nil
}

func (campaignListStub) AssignCampaignOwner(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (campaignListStub) ListCampaignEvents(context.Context, uuid.UUID, int32, int32) ([]adminapi.CampaignEventDTO, int64, error) {
	return nil, 0, nil
}

func (campaignListStub) BlockCampaignPlacement(context.Context, uuid.UUID, string) error {
	return nil
}

func mapPublisherTestError(err error) (int, string, string) {
	if errors.Is(err, controlplane.ErrPublisherScopeRequired) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL", "internal error"
}
