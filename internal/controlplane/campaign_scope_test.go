package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCampaignList_MediaBuyerScope(t *testing.T) {
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
	require.NoError(t, svc.CreateCustomer(ctx, custID, "TeamCo", 1_000_000, "USD"))

	buyerA := uuid.New()
	buyerB := uuid.New()
	campA := uuid.New()
	campB := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id, budget_limit, current_spend, owner_user_id)
		VALUES ($1, 'owned', 'ACTIVE', $4, 1000000, 0, $2),
		       ($3, 'other', 'ACTIVE', $4, 1000000, 0, $5)`,
		campA, buyerA, campB, custID, buyerB)
	require.NoError(t, err)

	token, err := tokenMaker.CreateToken(buyerA, uuid.New(), RoleMediaBuyer, custID, time.Hour)
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns?limit=50", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, int64(1), body.Total)
	require.Len(t, body.Items, 1)
	require.Equal(t, campA.String(), body.Items[0].ID)

	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campB.String(), http.NoBody)
	reqGet.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)
	require.Equal(t, http.StatusForbidden, recGet.Code)

	patchBody := `{"name":"hijack"}`
	reqPatch, _ := http.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campB.String(), strings.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	recPatch := httptest.NewRecorder()
	mux.ServeHTTP(recPatch, reqPatch)
	require.Equal(t, http.StatusForbidden, recPatch.Code)
}
