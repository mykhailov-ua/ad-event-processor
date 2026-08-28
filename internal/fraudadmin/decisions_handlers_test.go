package fraudadmin_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fraudDecisionsStub struct {
	customerID uuid.UUID
	ipHash     string
	hours      int
	campaignID *uuid.UUID
	response   fraudadmin.FraudDecisionDTO
	err        error
}

func (s *fraudDecisionsStub) ExplainFraudDecision(_ context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (fraudadmin.FraudDecisionDTO, error) {
	s.customerID = customerID
	s.ipHash = ipHash
	s.hours = hours
	s.campaignID = campaignID
	if s.err != nil {
		return fraudadmin.FraudDecisionDTO{}, s.err
	}
	return s.response, nil
}

func newFraudDecisionsHandlers(stub *fraudDecisionsStub) *fraudadmin.HTTPHandlers {
	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	return &fraudadmin.HTTPHandlers{
		Decisions: stub,
		AllowFraudDecision: func(string) bool {
			return true
		},
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
		WriteServiceError: func(w http.ResponseWriter, err error) {
			if errors.Is(err, fraudadmin.ErrFraudDecisionNotFound) {
				httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
				return
			}
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		},
	}
}

func TestGetFraudDecision_returnsBreakdown(t *testing.T) {
	ipHash := strings.Repeat("c", 32)
	stub := &fraudDecisionsStub{
		response: fraudadmin.FraudDecisionDTO{
			IPHash:              ipHash,
			CampaignID:          "660e8400-e29b-41d4-a716-446655440001",
			Tier:                "ivt",
			Score:               72,
			MLProbability:       0.81,
			AdjustedProbability: 0.81,
			ResidentialProxy:    true,
			Disclaimer:          fraudadmin.DecisionDisclaimer,
			Features:            map[string]float64{"events": 120},
			CampaignThresholds: fraudadmin.FraudTierThresholdsDTO{
				Scope:      "campaign",
				PassMax:    30,
				SuspectMax: 60,
				IVTMax:     80,
				BlockAbove: 100,
			},
		},
	}
	h := newFraudDecisionsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/decisions?customer_id=550e8400-e29b-41d4-a716-446655440000&ip_hash="+ipHash+"&hours=48", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, ipHash, stub.ipHash)
	require.Equal(t, 48, stub.hours)

	var out fraudadmin.FraudDecisionDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, "ivt", out.Tier)
	require.Equal(t, 72, out.Score)
	require.True(t, out.ResidentialProxy)
	require.Contains(t, out.Disclaimer, "last scorer window")
}

func TestGetFraudDecision_notFound(t *testing.T) {
	stub := &fraudDecisionsStub{err: fraudadmin.ErrFraudDecisionNotFound}
	h := newFraudDecisionsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	ipHash := strings.Repeat("d", 32)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/decisions?customer_id=550e8400-e29b-41d4-a716-446655440000&ip_hash="+ipHash, http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetFraudDecision_rejectsInvalidHash(t *testing.T) {
	stub := &fraudDecisionsStub{}
	h := newFraudDecisionsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/decisions?customer_id=550e8400-e29b-41d4-a716-446655440000&ip_hash=bad", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
