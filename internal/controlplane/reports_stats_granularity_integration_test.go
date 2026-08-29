package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetCampaignStats_DayGranularityRequiresOps(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Day Stats", 0, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Day Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "day-camp-1",
	})
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String()+"/stats?granularity=day", http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, custID)
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
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Day Admin", 0, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), campaign.CreateCampaignSpec{
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

	var report campaign.CampaignStatsDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&report))
	assert.Equal(t, "day", report.Granularity)
	assert.NotNil(t, report.Daily)
}
