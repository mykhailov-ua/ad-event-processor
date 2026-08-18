package controlplane_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fraudOverridesStub struct {
	customerID uuid.UUID
	lastReq    controlplane.FraudOverrideRequest
	called     bool
}

func (s *fraudOverridesStub) ApplyFraudScoringOverrideForCustomer(_ context.Context, customerID uuid.UUID, req controlplane.FraudOverrideRequest) error {
	s.customerID = customerID
	s.lastReq = req
	s.called = true
	return nil
}

func newFraudOverridesHandlers(stub *fraudOverridesStub) *controlplane.FraudHTTPHandlers {
	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	return &controlplane.FraudHTTPHandlers{
		Overrides: stub,
		RequireAnyPermission: func(required []string, next http.HandlerFunc) http.HandlerFunc {
			perms := map[string]bool{
				"audit:write":     true,
				"campaigns:write": true,
				"shards:write":    true,
			}
			return func(w http.ResponseWriter, r *http.Request) {
				for _, p := range required {
					if perms[p] {
						next(w, r)
						return
					}
				}
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			}
		},
		ResolveCustomerID: func(_ *http.Request, hint *uuid.UUID) (uuid.UUID, error) {
			if hint != nil && *hint != uuid.Nil {
				return *hint, nil
			}
			return customerID, nil
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapPublisherTestError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
}

func TestPostFraudOverride_acceptsCampaignAndIPHash(t *testing.T) {
	stub := &fraudOverridesStub{}
	h := newFraudOverridesHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	campaignID := uuid.New()
	body := `{"campaign_id":"` + campaignID.String() + `","ip_hash":"0123456789abcdef0123456789abcdef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/overrides?customer_id="+customerID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.True(t, stub.called)
	require.Equal(t, customerID, stub.customerID)
	require.NotNil(t, stub.lastReq.CampaignID)
	require.Equal(t, campaignID.String(), *stub.lastReq.CampaignID)
	require.NotNil(t, stub.lastReq.IPHash)
}

func TestPostFraudOverride_invalidJSON(t *testing.T) {
	stub := &fraudOverridesStub{}
	h := newFraudOverridesHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/overrides?customer_id="+customerID.String(), strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
