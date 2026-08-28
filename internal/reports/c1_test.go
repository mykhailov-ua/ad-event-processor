package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostbackReconcileStatus_holdoutMissingDispatch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "missing_postback", postbackReconcileStatus(""))
	assert.Equal(t, "ok", postbackReconcileStatus("SENT"))
	assert.Equal(t, "FAILED", postbackReconcileStatus("FAILED"))
}

func TestCampaignDayKey_format(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	day := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111|2026-03-15", campaignDayKey(id, day))
}

func TestMeasuredPacingDriftPct_holdoutOverDelivery(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.25, measuredPacingDriftPct(100, 125), 0.0001)
	assert.InDelta(t, -0.5, measuredPacingDriftPct(100, 50), 0.0001)
}

func TestCalcSilentRejectRatio_holdout(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.2, calcSilentRejectRatio(2, 10), 0.0001)
}

func TestCalcRtbWinRate_holdout(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.25, calcRtbWinRate(25, 100), 0.0001)
}

func TestReports_C1RoutesRegistered(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	paths := []string{
		"/api/v1/reports/filter-rejects",
		"/api/v1/reports/rtb/overview",
		"/api/v1/reports/rtb/no-bid-reasons",
		"/api/v1/reports/rtb/geo-device",
		"/api/v1/reports/postback-reconciliation?customer_id=" + uuid.New().String(),
		"/api/v1/reports/pacing-drift?customer_id=" + uuid.New().String(),
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusServiceUnavailable, w.Code, "path=%s", path)
	}
}
