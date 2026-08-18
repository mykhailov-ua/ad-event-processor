package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fraudIntegrationsStub struct {
	customerID uuid.UUID
	rows       []controlplane.FraudIntegrationDTO
}

func (s *fraudIntegrationsStub) ListFraudIntegrationsForCustomer(_ context.Context, customerID uuid.UUID) ([]controlplane.FraudIntegrationDTO, error) {
	s.customerID = customerID
	return s.rows, nil
}

func newFraudIntegrationsHandlers(stub *fraudIntegrationsStub) *controlplane.FraudHTTPHandlers {
	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	return &controlplane.FraudHTTPHandlers{
		Integrations: stub,
		RequirePermission: func(permission string, next http.HandlerFunc) http.HandlerFunc {
			if permission != "audit:read" {
				return func(w http.ResponseWriter, _ *http.Request) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
				}
			}
			return next
		},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return customerID, nil
		},
	}
}

func TestListFraudIntegrations_customerScoped(t *testing.T) {
	stub := &fraudIntegrationsStub{
		rows: []controlplane.FraudIntegrationDTO{
			{
				CampaignID:   "660e8400-e29b-41d4-a716-446655440001",
				Name:         "US Push",
				Provider:     "facebook",
				Configured:   true,
				HealthStatus: "failing",
				DLQCount:     3,
				LastError:    "HTTP 401",
			},
		},
	}
	h := newFraudIntegrationsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/integrations?customer_id=550e8400-e29b-41d4-a716-446655440000", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out []controlplane.FraudIntegrationDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 1)
	require.Equal(t, "failing", out[0].HealthStatus)
	require.Equal(t, stub.customerID, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
}
