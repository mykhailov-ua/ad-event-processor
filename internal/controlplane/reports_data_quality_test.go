package controlplane

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataQualitySeverity_holdoutAboveTolerance(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "medium", dataQualitySeverity(1000, hyg30CHStatsTolerancePct+0.001))
	assert.Equal(t, "ok", dataQualitySeverity(1000, hyg30CHStatsTolerancePct))
	assert.Equal(t, "high", dataQualitySeverity(100, 0.10))
	assert.Equal(t, "info", dataQualitySeverity(0, 1.0))
}

func TestCampaignDailyTotalKey_format(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	day := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111|2026-03-15", campaignDailyTotalKey(id, day))
}

func TestDataQualityDiffPct_holdoutDetectsDrift(t *testing.T) {
	t.Parallel()

	pgTotal := int64(100)
	chTotal := int64(50)
	diffPct := math.Abs(float64(chTotal-pgTotal)) / float64(pgTotal)
	assert.InDelta(t, 0.5, diffPct, 0.0001)
	assert.Equal(t, "high", dataQualitySeverity(pgTotal, diffPct))
}

func TestReports_DataQualityRouteRegistered(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/data-quality?customer_id="+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
