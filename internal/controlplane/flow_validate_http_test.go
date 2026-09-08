package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFlowValidateHTTP_weightSum400_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: flow validate HTTP returns 400 on weight sum drift")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, _ := integrationTestAuth(t, redisClient, cfg)
	authMW.SetPolicyStore(ctrlhttp.InitPolicyStore())
	authMW.SetPool(pool)

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	landerID := uuid.New()
	offerID := uuid.New()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'validate-lander', 'https://l.test')`, landerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO offers (id, name, url) VALUES ($1, 'validate-offer', 'https://o.test')`, offerID)
	require.NoError(t, err)

	payload, err := json.Marshal(map[string]any{
		"paths": []map[string]any{
			{
				"weight":  50,
				"landers": []map[string]any{{"lander_id": landerID.String(), "weight": 100}},
				"offers":  []map[string]any{{"offer_id": offerID.String(), "weight": 100}},
			},
			{
				"weight":  49,
				"landers": []map[string]any{{"lander_id": landerID.String(), "weight": 100}},
				"offers":  []map[string]any{{"offer_id": offerID.String(), "weight": 100}},
			},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows/validate", bytes.NewReader(payload))
	withAdminAPIKey(req, cfg)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	var body struct {
		Valid      bool `json:"valid"`
		PathErrors []struct {
			Code string `json:"code"`
		} `json:"path_errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.False(t, body.Valid)
	require.NotEmpty(t, body.PathErrors)
	require.Equal(t, "weight_sum", body.PathErrors[0].Code)
}
