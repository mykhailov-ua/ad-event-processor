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
	"ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_RBACMaskEnforced(t *testing.T) {
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
	authMW.SetPolicyStore(InitPolicyStore())
	authMW.SetPool(pool)

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "BuyerCo", 1_000_000, "USD"))

	campID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, target_url, creative_payload, referrer_filter)
		VALUES ($1, 'masked-camp', 'ACTIVE', $2, 1000000, 0, 'https://secret.example/path', '{"type":"image"}', 'ref.example')`,
		campID, custID)
	require.NoError(t, err)

	buyerID := uuid.New()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String(), http.NoBody)
	withSessionUser(req, tokenMaker, RoleBuyer, custID)
	token, err := tokenMaker.CreateToken(buyerID, uuid.New(), RoleBuyer, custID, time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	_, hasURL := body["target_url"]
	require.False(t, hasURL, "buyer must not receive target_url")

	faultproof.Log(t, "rbac_mask_enforced", map[string]string{
		"fault":       "rbac_mask_enforced",
		"campaign_id": campID.String(),
		"role":        RoleBuyer,
	})
}

func TestAPI_GetCampaign_BuyerMasking(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, tokenMaker := integrationTestAuth(t, rdb, cfg)
	authMW.SetPolicyStore(InitPolicyStore())
	authMW.SetPool(pool)

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Adv", 1_000_000, "USD"))
	campID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, target_url)
		VALUES ($1, 'c', 'ACTIVE', $2, 1000000, 0, 'https://secret.example')`, campID, custID)
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campID.String(), http.NoBody)
	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), RoleBuyer, custID, time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &raw))
	_, ok := raw["target_url"]
	require.False(t, ok)
}
