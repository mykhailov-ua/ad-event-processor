package reports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatsQuery_dayRequiresOps(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stats?granularity=day", http.NoBody)
	_, _, _, err := parseStatsQuery(req, false)
	require.Error(t, err)
}

func TestParseStatsQuery_dayAllowedForOps(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stats?granularity=day", http.NoBody)
	_, _, granularity, err := parseStatsQuery(req, true)
	require.NoError(t, err)
	assert.Equal(t, "day", granularity)
}

func TestParseStatsQuery_dayRangeCap365(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/stats?granularity=day&from="+from+"&to="+to, http.NoBody)
	_, _, _, err := parseStatsQuery(req, true)
	require.Error(t, err)
}

func TestParseStatsQuery_unknownGranularity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stats?granularity=week", http.NoBody)
	_, _, _, err := parseStatsQuery(req, true)
	require.Error(t, err)
}

type stubCampaignStatsReader struct{}

func (stubCampaignStatsReader) GetCampaignStats(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) (CampaignStatsDTO, error) {
	return CampaignStatsDTO{}, nil
}

func TestGetCampaignStatsHTTP_rejectsUnknownGranularity(t *testing.T) {
	reports := &ReportsHTTPHandlers{
		CampaignStats: stubCampaignStatsReader{},
		ApplyRateLimit: func(next http.HandlerFunc) http.HandlerFunc {
			return next
		},
		RequireAnyPermission: func(_ []string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
	}
	mux := http.NewServeMux()
	reports.registerCampaignStats(mux)

	campID := uuid.New()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/stats?granularity=week", http.NoBody)
	ctx := authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{permShardsRead: {}},
	})
	req = req.WithContext(ctx)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var body httpresponse.ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "BAD_REQUEST", body.Error.Code)
}

func TestRequestHasShardsRead_adminSnapshot(t *testing.T) {
	h := &ReportsHTTPHandlers{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	ctx := authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{permShardsRead: {}},
	})
	req = req.WithContext(ctx)
	assert.True(t, h.requestHasShardsRead(req))
}

func TestRequestHasShardsRead_buyerDenied(t *testing.T) {
	h := &ReportsHTTPHandlers{}
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	ctx := authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{"campaigns:read": {}},
	})
	req = req.WithContext(ctx)
	assert.False(t, h.requestHasShardsRead(req))
}
