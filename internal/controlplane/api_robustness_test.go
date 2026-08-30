package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/platformadmin"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestManagementAPI_Robustness(t *testing.T) {
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

	t.Run("InvalidInput_MalformedJSON", func(t *testing.T) {
		endpoints := []struct {
			method string
			path   string
		}{
			{"POST", "/api/v1/forecast/campaign"},
			{"POST", "/api/v1/consent"},
			{"POST", "/api/v1/ops/blacklist"},
		}

		for _, tc := range endpoints {
			req, _ := http.NewRequest(tc.method, tc.path, bytes.NewBufferString("{malformed-json"))
			req.Header.Set("X-Admin-API-Key", "test-secret")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusBadRequest, resp.Code, "expected 400 for %s %s", tc.method, tc.path)
		}
	})

	t.Run("InvalidInput_InvalidUUID_URL", func(t *testing.T) {
		paths := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/campaigns/invalid-uuid"},
			{"GET", "/api/v1/campaigns/invalid-uuid/stats"},
			{"GET", "/api/v1/customers/invalid-uuid/balance"},
		}

		for _, tc := range paths {
			req, _ := http.NewRequest(tc.method, tc.path, http.NoBody)
			req.Header.Set("X-Admin-API-Key", "test-secret")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusBadRequest, resp.Code, "expected 400 for %s %s", tc.method, tc.path)
		}
	})

	t.Run("DBFailure_Simulation", func(t *testing.T) {
		badPool, cleanupBadDB := database.SetupTestDB(t)
		cleanupBadDB()

		badSvc := NewService(context.Background(), badPool, []redis.UniversalClient{redisClient}, nil, cfg)
		defer badSvc.Close()
		badH := NewHandler(badSvc, cfg, nil, nil, nil, nil)
		badMux := http.NewServeMux()
		badH.RegisterRoutes(badMux)

		paths := []string{
			"/api/v1/audit",
			"/api/v1/ops/shards",
		}

		for _, path := range paths {
			req, _ := http.NewRequest("GET", path, http.NoBody)
			req.Header.Set("X-Admin-API-Key", "test-secret")
			resp := httptest.NewRecorder()
			badMux.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusInternalServerError, resp.Code, "expected 500 for DB failure GET %s", path)
		}
	})

	t.Run("BackgroundSync_GoroutineLifecycle", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			platformadmin.NewSystemStateWorker(svc).Start(ctx)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()
	})

	t.Run("LegacyAdmin_Gone", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/customers", http.NoBody)
		req.Header.Set("X-Admin-API-Key", "test-secret")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusGone, resp.Code)
	})
}
