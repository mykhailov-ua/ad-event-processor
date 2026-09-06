package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

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
	queryBudgetListCampaigns        = 8
	queryBudgetGetCampaign          = 6
	queryBudgetListCustomers        = 12
	queryBudgetExportCampaignsBatch = 12
	queryBudgetBulkPauseCampaign    = 24
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

func TestQueryBudget_ExportCampaignsBatch_HTTP(t *testing.T) {
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
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Export Batch Corp", 500_000_000, "USD"))

	campIDs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
			CustomerID:       custID,
			Name:             fmt.Sprintf("Export Batch Camp %d", i),
			BudgetLimitMicro: 10_000_000,
			PacingMode:       "ASAP",
			Timezone:         "UTC",
			IdempotencyKey:   fmt.Sprintf("qb-export-batch-%d", i),
		})
		require.NoError(t, err)
		campIDs = append(campIDs, campID.String())
	}

	counter.Reset()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/export?ids="+strings.Join(campIDs, ","), http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp campaign.CampaignExportBatchResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Items, 5)

	queries := counter.Snapshot()
	t.Logf("export campaigns batch HTTP queries=%d budget<=%d campaigns=%d", queries, queryBudgetExportCampaignsBatch, len(campIDs))
	assert.LessOrEqual(t, queries, int64(queryBudgetExportCampaignsBatch), "N+1 regression: batch export must not scale linearly with campaign count")
}

func TestQueryBudget_BulkPauseCampaign_HTTP(t *testing.T) {
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
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Bulk Pause Corp", 500_000_000, "USD"))

	campIDs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
			CustomerID:       custID,
			Name:             fmt.Sprintf("Bulk Pause Camp %d", i),
			BudgetLimitMicro: 10_000_000,
			PacingMode:       "ASAP",
			Timezone:         "UTC",
			IdempotencyKey:   fmt.Sprintf("qb-bulk-pause-%d", i),
		})
		require.NoError(t, err)
		campIDs = append(campIDs, campID.String())
	}

	body, err := json.Marshal(map[string]any{
		"action":       "pause",
		"campaign_ids": campIDs,
	})
	require.NoError(t, err)

	counter.Reset()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-action", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	withSessionUser(req, tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Results []struct {
			ID  string `json:"id"`
			OK  bool   `json:"ok"`
			Err string `json:"error_code,omitempty"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 5)
	for _, row := range resp.Results {
		assert.True(t, row.OK, row.ID)
	}

	queries := counter.Snapshot()
	t.Logf("bulk pause HTTP queries=%d budget<=%d campaigns=%d", queries, queryBudgetBulkPauseCampaign, len(campIDs))
	assert.LessOrEqual(t, queries, int64(queryBudgetBulkPauseCampaign), "N+1 regression: bulk pause must use one PG txn, not N")
}
