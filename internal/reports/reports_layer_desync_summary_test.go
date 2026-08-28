package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLayerDesyncSummaryQuery_groupsByCampaignAndCount_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, layerDesyncSummaryQuery, "FROM fraud_events")
	require.Contains(t, layerDesyncSummaryQuery, "layer_desync_count")
	require.Contains(t, layerDesyncSummaryQuery, "GROUP BY campaign_id, layer_desync_count")
	require.Contains(t, layerDesyncSummaryQuery, "countIf(silent_reject_event = 1)")
	require.Contains(t, layerDesyncSummaryCountQuery, "GROUP BY campaign_id, layer_desync_count")
}

func TestLayerDesyncSummaryRoute_clickhouseUnavailable503(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/layer-desync-summary?customer_id="+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "CLICKHOUSE_UNAVAILABLE")
}
