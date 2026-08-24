package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fraudLabelsStub struct {
	customerID uuid.UUID
	labels     []controlplane.MLManualLabelDTO
	lastUpsert controlplane.FraudManualLabelRequest
	bulkRows   []controlplane.FraudManualLabelRow
}

func (s *fraudLabelsStub) ListMLManualLabelsForCustomer(_ context.Context, customerID uuid.UUID, limit int) ([]controlplane.MLManualLabelDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	out := s.labels
	if len(out) > limit {
		out = out[:limit]
	}
	s.customerID = customerID
	return out, nil
}

func (s *fraudLabelsStub) UpsertMLManualLabelForCustomer(_ context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	s.customerID = customerID
	s.lastUpsert = controlplane.FraudManualLabelRequest{IPHash: ipHash, Label: label, Reason: reason}
	return nil
}

func (s *fraudLabelsStub) BulkUpsertMLManualLabelsForCustomer(_ context.Context, customerID uuid.UUID, rows []controlplane.FraudManualLabelRow) (int, error) {
	s.customerID = customerID
	s.bulkRows = rows
	return len(rows), nil
}

func newFraudLabelsHandlers(stub *fraudLabelsStub) *controlplane.FraudHTTPHandlers {
	customerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	return &controlplane.FraudHTTPHandlers{
		Labels: stub,
		RequirePermission: func(permission string, next http.HandlerFunc) http.HandlerFunc {
			if permission != "audit:read" {
				httpresponse.Error(nil, http.StatusForbidden, "FORBIDDEN", "forbidden")
				return func(w http.ResponseWriter, _ *http.Request) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
				}
			}
			return next
		},
		RequireAnyPermission: func(required []string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return customerID, nil
		},
	}
}

func TestListFraudLabels_customerScoped(t *testing.T) {
	stub := &fraudLabelsStub{
		labels: []controlplane.MLManualLabelDTO{
			{IPHash: strings.Repeat("a", 32), Label: 1, Reason: "bot", Source: "admin_ui"},
		},
	}
	h := newFraudLabelsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/labels?customer_id=550e8400-e29b-41d4-a716-446655440000", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out []controlplane.MLManualLabelDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 1)
	require.Equal(t, stub.customerID, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
}

func TestPostFraudLabel_upserts(t *testing.T) {
	stub := &fraudLabelsStub{}
	h := newFraudLabelsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	ipHash := strings.Repeat("b", 32)
	body := `{"ip_hash":"` + ipHash + `","label":1,"reason":"click farm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/labels?customer_id=550e8400-e29b-41d4-a716-446655440000", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, ipHash, stub.lastUpsert.IPHash)
	require.Equal(t, 1, stub.lastUpsert.Label)
}

func TestPostFraudLabelsBulk_rejectsEmpty(t *testing.T) {
	stub := &fraudLabelsStub{}
	h := newFraudLabelsHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/labels/bulk?customer_id=550e8400-e29b-41d4-a716-446655440000", strings.NewReader(`{"rows":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
