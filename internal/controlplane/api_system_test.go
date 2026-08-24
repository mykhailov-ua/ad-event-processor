package controlplane

import (
	"bytes"
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

func TestManagementAPI_System(t *testing.T) {
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

	ctx := context.Background()

	t.Run("SettingsCycle", func(t *testing.T) {
		settings := map[string]string{
			"rate_limit_per_min": "100",
			"click_amount":       "0.05",
		}
		require.NoError(t, svc.UpdateSettings(ctx, settings))

		got, err := svc.GetSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, "100", got["rate_limit_per_min"])
		assert.Equal(t, "0.05", got["click_amount"])

		assert.Eventually(t, func() bool {
			val, err := redisClient.HGet(ctx, "config:values", "rate_limit_per_min").Result()
			return err == nil && val == "100"
		}, 2*time.Second, 20*time.Millisecond)
	})

	t.Run("BlacklistCycle", func(t *testing.T) {
		require.NoError(t, svc.BlockIPWithTTL(ctx, "192.168.1.50", "fraud", nil))

		assert.Eventually(t, func() bool {
			isMember, err := redisClient.SIsMember(ctx, "blacklist:fraud", "192.168.1.50").Result()
			return err == nil && isMember
		}, 2*time.Second, 20*time.Millisecond)

		reqList, _ := http.NewRequest("GET", "/api/v1/ops/blacklist", http.NoBody)
		reqList.Header.Set("X-Admin-API-Key", "test-secret")
		respList := httptest.NewRecorder()
		mux.ServeHTTP(respList, reqList)
		assert.Equal(t, http.StatusOK, respList.Code)
		assert.NotEmpty(t, respList.Header().Get("X-Total-Count"))

		var bl []BlacklistDTO
		err := json.NewDecoder(respList.Body).Decode(&bl)
		require.NoError(t, err)
		require.NotEmpty(t, bl)
		assert.Equal(t, "192.168.1.50", bl[0].IP)
		assert.Equal(t, "fraud", bl[0].Reason)

		require.NoError(t, svc.SyncSystemState(ctx))

		body, _ := json.Marshal(map[string]string{"ip": "192.168.1.50", "source": "fraud"})
		reqDel, _ := http.NewRequest("DELETE", "/api/v1/ops/blacklist", bytes.NewReader(body))
		reqDel.Header.Set("X-Admin-API-Key", "test-secret")
		respDel := httptest.NewRecorder()
		mux.ServeHTTP(respDel, reqDel)
		assert.Equal(t, http.StatusNoContent, respDel.Code)

		assert.Eventually(t, func() bool {
			isMember, err := redisClient.SIsMember(ctx, "blacklist:fraud", "192.168.1.50").Result()
			return err == nil && !isMember
		}, 2*time.Second, 20*time.Millisecond)
	})
}
