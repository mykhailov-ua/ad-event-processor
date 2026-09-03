package campaign

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListCampaignMetricsTotals_requiresPostgres(t *testing.T) {
	h := &CampaignsHTTPHandlers{}
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/campaigns/metrics-totals?from="+time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339)+
			"&to="+time.Now().UTC().Format(time.RFC3339),
		nil,
	)
	rec := httptest.NewRecorder()

	h.listCampaignMetricsTotals(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
