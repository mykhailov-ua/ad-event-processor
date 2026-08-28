package controlplane_test

import (
	"bytes"
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

type selfServeTemplatesStub struct{}

func (s selfServeTemplatesStub) ListCampaignTemplates(_ context.Context, _ uuid.UUID, _, _ int32) ([]controlplane.CampaignTemplateDTO, int64, error) {
	return nil, 0, nil
}

func (s selfServeTemplatesStub) CreateCampaignFromTemplate(_ context.Context, _, _ uuid.UUID, _ string, _ *int64, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func TestSelfServeCreateCampaign_requiresTemplateID(t *testing.T) {
	h := &controlplane.SelfServeHTTPHandlers{
		Templates: selfServeTemplatesStub{},
		ResolveSelfServeCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, err := json.Marshal(map[string]any{"name": "No template"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/campaigns", bytes.NewReader(body))
	req.Header.Set("Idempotency-Key", "test-key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSelfServeListTemplates_operatorPermBlocked(t *testing.T) {
	h := &controlplane.SelfServeHTTPHandlers{
		Templates: selfServeTemplatesStub{},
		RequireAnyPermission: func(_ []string, next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			}
		},
		ResolveSelfServeCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/selfserve/templates", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}
