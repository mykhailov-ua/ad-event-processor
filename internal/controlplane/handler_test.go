package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementAPI_Hardening(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey: "test-secret",
	}

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("LegacyAdmin_Gone", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/settings", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusGone, resp.Code)
	})

	t.Run("UpdateSettings_Success", func(t *testing.T) {
		require.NoError(t, svc.UpdateSettings(context.Background(), map[string]string{"rate_limit_per_min": "500"}))
		assert.Eventually(t, func() bool {
			v, _ := redisClient.Get(context.Background(), "config:version").Int64()
			return v == int64(1)
		}, 2*time.Second, 20*time.Millisecond)
	})

	t.Run("Blacklist_Success", func(t *testing.T) {
		require.NoError(t, svc.BlockIPWithTTL(context.Background(), "9.9.9.9", "manual", nil))
		assert.Eventually(t, func() bool {
			isMember, _ := redisClient.SIsMember(context.Background(), "blacklist:manual", "9.9.9.9").Result()
			return isMember
		}, 2*time.Second, 20*time.Millisecond)
	})

	t.Run("ListAudit", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/audit?limit=10", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))

		var logs []any
		err := json.NewDecoder(resp.Body).Decode(&logs)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 0)
	})

	t.Run("RateLimit", func(t *testing.T) {
		for range 60 {
			req, _ := http.NewRequest("GET", "/api/v1/audit", http.NoBody)
			req.Header.Set("X-Admin-API-Key", "test-secret")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			if resp.Code == http.StatusTooManyRequests {
				return
			}
		}
		t.Errorf("Rate limiter did not trigger after 60 requests")
	})
}
