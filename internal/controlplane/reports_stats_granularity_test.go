package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func TestGetCampaignStatsHTTP_rejectsUnknownGranularity(t *testing.T) {
	reports := &ReportsHTTPHandlers{
		CampaignStats: &Service{},
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
		Permissions: map[string]struct{}{PermShardsRead: {}},
	})
	req = req.WithContext(ctx)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var body httpresponse.ErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "BAD_REQUEST", body.Error.Code)
}

func TestAPI_GetCampaignStats_DayGranularityRequiresOps(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, tokenMaker := integrationTestAuth(t, rdb, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Day Stats", 0, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Day Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "day-camp-1",
	})
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/stats?granularity=day", http.NoBody)
	withSessionUser(req, tokenMaker, RoleUser, custID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestAPI_GetCampaignStats_DayGranularityAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Day Admin", 0, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Day Admin Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "day-admin-1",
	})
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/stats?granularity=day", http.NoBody)
	withAdminAPIKey(req, cfg)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	var report CampaignStatsDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&report))
	assert.Equal(t, "day", report.Granularity)
	assert.NotNil(t, report.Daily)
}

func TestRequestHasShardsRead_adminSnapshot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	ctx := authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{PermShardsRead: {}},
	})
	req = req.WithContext(ctx)
	assert.True(t, requestHasShardsRead(req))
}

func TestRequestHasShardsRead_buyerDenied(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	ctx := authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{PermCampaignsRead: {}},
	})
	req = req.WithContext(ctx)
	assert.False(t, requestHasShardsRead(req))
}
