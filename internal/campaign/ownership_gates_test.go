package campaign

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"ad-event-processor/pkg/httpresponse"
)

func TestCampaignOwnerUserFilter_mediaBuyerScoped(t *testing.T) {
	t.Parallel()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:   authz.RoleMediaBuyer,
		UserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	})
	filter := campaignOwnerUserFilter(ctx)
	assert.True(t, filter.Valid)
}

func TestCampaignOwnerUserFilter_adminUnscoped(t *testing.T) {
	t.Parallel()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:   authz.RoleAdmin,
		UserID: uuid.New(),
	})
	filter := campaignOwnerUserFilter(ctx)
	assert.False(t, filter.Valid)
}

func TestIntegrationPanel_forbiddenCampaign404(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	h := &CampaignsHTTPHandlers{
		Campaigns: &patchRevisionCampaignStub{},
		AuthorizeCampaignAccess: func(_ *http.Request, id uuid.UUID) error {
			if id == campaignID {
				return ErrCampaignNotFound
			}
			return nil
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapServiceError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
	mux := http.NewServeMux()
	h.registerCampaignEditorRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaignID.String()+"/integration-panel", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
