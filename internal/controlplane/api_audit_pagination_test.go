package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditPaginationMarker struct {
	I int `json:"i"`
}

func TestHandlerAudit_pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()

	ctx := context.Background()
	adminID := uuid.New()
	for i := range 55 {
		svc.AuditLog(ctx, nil, adminID, "PAGINATION_TEST", "system", nil, auditPaginationMarker{I: i}, nil)
	}

	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("default_limit_50", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/audit", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "55", resp.Header().Get("X-Total-Count"))

		var items []AuditLogDTO
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&items))
		assert.Len(t, items, 50)
	})

	t.Run("limit_and_offset", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/audit?limit=10&offset=50", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "55", resp.Header().Get("X-Total-Count"))

		var items []AuditLogDTO
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&items))
		assert.Len(t, items, 5)
	})

	t.Run("limit_capped_at_1000", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/audit?limit=1000000", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var items []AuditLogDTO
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&items))
		assert.Len(t, items, 55)
	})
}
