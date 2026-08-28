package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"bytes"
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

func TestFault_SelfServeIdempotentCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:             "test-secret",
		TokenSymmetricKey:       "01234567890123456789012345678901",
		SelfServeBudgetMinMicro: 1_000_000,
		SelfServeBudgetMaxMicro: 100_000_000_000,
	}
	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Idem", 100_000_000, "USD"))

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"name":               "Idempotent",
		"budget_limit_micro": int64(5_000_000),
	})
	idemKey := "ss-idem-fault-" + uuid.New().String()

	var firstID string
	for i := range 2 {
		req, _ := http.NewRequest("POST", "/api/v1/selfserve/campaigns", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)
		withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, custID)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code, "attempt %d", i+1)
		var resp map[string]any
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		id, ok := resp["id"].(string)
		require.True(t, ok)
		if i == 0 {
			firstID = id
		} else {
			assert.Equal(t, firstID, id)
		}
	}
}
