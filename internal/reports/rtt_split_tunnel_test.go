package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRTTSplitTunnelQuery_usesImpressionsColumn_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, rttSplitTunnelQuery, "FROM impressions")
	require.Contains(t, rttSplitTunnelQuery, "rtt_split_delta_ms")
	require.Contains(t, rttSplitTunnelQuery, "rtt_syn_ms")
}

func TestRTTSplitTunnelRoute_clickhouseUnavailable503(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/rtt-split-tunnel?customer_id="+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
