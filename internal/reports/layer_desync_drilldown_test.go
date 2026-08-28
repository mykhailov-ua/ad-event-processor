package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLayerDesyncDrilldownQuery_groupsByFraudReason_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, layerDesyncDrilldownRowsQuery, "FROM fraud_events")
	require.Contains(t, layerDesyncDrilldownRowsQuery, "layer_desync_count")
	require.Contains(t, layerDesyncDrilldownRowsQuery, "GROUP BY fraud_reason")
	require.Contains(t, layerDesyncDrilldownSeriesQuery, "toStartOfHour(created_at)")
}

func TestLayerDesyncDrilldownRoute_clickhouseUnavailable503(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/layer-desync-drilldown?customer_id="+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "CLICKHOUSE_UNAVAILABLE")
}
