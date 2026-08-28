package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/platformadmin"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	queryBudgetListCampaigns = 8
	queryBudgetGetCampaign   = 6
	queryBudgetListCustomers = 12
)

func TestQueryBudget_ListCampaigns_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
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

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Query Budget Corp", 100_000_000, "USD"))
	_, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Budget Camp",
		BudgetLimitMicro: 50_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "qb-list-camp-1",
	})
	require.NoError(t, err)

	counter.Reset()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns?customer_id="+custID.String()+"&limit=50", http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	queries := counter.Snapshot()
	t.Logf("list campaigns HTTP queries=%d budget<=%d", queries, queryBudgetListCampaigns)
	assert.LessOrEqual(t, queries, int64(queryBudgetListCampaigns), "N+1 regression: list campaigns")
}

func TestQueryBudget_GetCampaign_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
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

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Query Budget Corp", 100_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Budget Camp",
		BudgetLimitMicro: 50_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "qb-get-camp-1",
	})
	require.NoError(t, err)

	counter.Reset()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String(), http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	queries := counter.Snapshot()
	t.Logf("get campaign HTTP queries=%d budget<=%d", queries, queryBudgetGetCampaign)
	assert.LessOrEqual(t, queries, int64(queryBudgetGetCampaign), "N+1 regression: get campaign")
}

func TestQueryBudget_ListCustomers_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
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

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Query Budget Corp", 100_000_000, "USD"))

	counter.Reset()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers?limit=50&offset=0", http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp platformadmin.CustomerListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Greater(t, resp.Total, int64(0))

	queries := counter.Snapshot()
	t.Logf("list customers HTTP queries=%d budget<=%d", queries, queryBudgetListCustomers)
	assert.LessOrEqual(t, queries, int64(queryBudgetListCustomers), "N+1 regression: list customers")
}
