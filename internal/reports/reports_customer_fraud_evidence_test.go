package reports

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestCustomerFraudEvidencePerms_buyerMaskedDenied_holdout(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/reports/customer-fraud-evidence", requireAnyPermissionFromSnapshot(reportPermsCustomerFraudEvidence(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customer-fraud-evidence", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{permCampaignsReadMasked: {}},
		Mask:        authz.MaskMasked,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestCustomerFraudEvidencePerms_fullCampaignReadAllowed(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/reports/customer-fraud-evidence", requireAnyPermissionFromSnapshot(reportPermsCustomerFraudEvidence(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customer-fraud-evidence", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{"campaigns:read": {}},
		Mask:        authz.MaskFull,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestCustomerFraudEvidenceRoute_maskedHandler403(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{
		FraudEvidencePackHMACSecret: []byte("secret"),
	}
	mux := http.NewServeMux()
	h.registerCustomerFraudEvidenceReport(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customer-fraud-evidence?customer_id=00000000-0000-0000-0000-000000000001&click_id=clk", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{"campaigns:read": {}},
		Mask:        authz.MaskMasked,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, resp.Code)
}
