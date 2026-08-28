package reports

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"

	"github.com/stretchr/testify/assert"
)

func requireAnyPermissionFromSnapshot(required []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap, ok := authz.SnapshotFromContext(r.Context())
		if !ok {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			return
		}
		for _, perm := range required {
			if snap.Has(perm) {
				next(w, r)
				return
			}
		}
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	}
}

func TestReportPermsFraudCustomer_includesMaskedRead(t *testing.T) {
	t.Parallel()
	perms := ReportPermsFraudCustomer()
	assert.Contains(t, perms, permCampaignsReadMasked)
}

func TestFraudEvidencePackPerms_buyerMaskedDenied(t *testing.T) {
	t.Parallel()
	check := func(perm string) bool {
		for _, p := range reportPermsFraudOperator {
			if p == perm {
				return true
			}
		}
		return false
	}
	assert.False(t, check(permCampaignsReadMasked))
}

func TestRequireAnyPermission_fraudEvidencePack_buyer403(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/fraud-evidence-pack", requireAnyPermissionFromSnapshot(perms, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/fraud-evidence-pack", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{permCampaignsReadMasked: {}},
		Mask:        authz.MaskMasked,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestRequireAnyPermission_filterRejects_buyer403(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	perms := []string{"audit:read"}
	mux.HandleFunc("GET /api/v1/reports/filter-rejects", requireAnyPermissionFromSnapshot(perms, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/filter-rejects", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{permCampaignsReadMasked: {}},
		Mask:        authz.MaskMasked,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestRequireAnyPermission_fraudBreakdown_buyer200(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/reports/fraud-breakdown", requireAnyPermissionFromSnapshot(ReportPermsFraudCustomer(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/fraud-breakdown", http.NoBody)
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{permCampaignsReadMasked: {}},
		Mask:        authz.MaskMasked,
	})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req.WithContext(ctx))
	assert.Equal(t, http.StatusOK, resp.Code)
}
