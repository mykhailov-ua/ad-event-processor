package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCampaignToggleCohort_invalidToggleField400(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/campaign-toggle-cohort?campaign_id="+uuid.New().String()+"&toggle_field=bad", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCampaignToggleCohort_missingCampaignID400(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/campaign-toggle-cohort?toggle_field=silent_reject_enabled", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
