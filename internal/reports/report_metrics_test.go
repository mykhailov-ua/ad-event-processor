package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveReportHandler_recordsBadRequest(t *testing.T) {
	t.Parallel()

	handler := ObserveReportHandler("placements", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/placements", http.NoBody)
	rec := httptest.NewRecorder()
	handler(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReportErrorReason_statusMapping(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ch_unavailable", reportErrorReason(http.StatusServiceUnavailable))
	assert.Equal(t, "forbidden", reportErrorReason(http.StatusForbidden))
	assert.Equal(t, "query_timeout", reportErrorReason(http.StatusGatewayTimeout))
	assert.Equal(t, "internal", reportErrorReason(http.StatusInternalServerError))
}

func TestLiveReportMetricKeys_includeExportAndOps(t *testing.T) {
	t.Parallel()
	keys := LiveReportMetricKeys()
	assert.Contains(t, keys, "placements")
	assert.Contains(t, keys, "edge-parity")
	assert.Contains(t, keys, "ml/score-distribution")
	assert.Contains(t, keys, "campaign-stats")
	assert.GreaterOrEqual(t, len(keys), len(liveReportExportKeys())+4)
}
