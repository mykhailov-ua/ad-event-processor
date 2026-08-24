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

func TestCustomersList_Handler(t *testing.T) {
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
	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "List Test Corp", 200_000_000, "USD"))
	require.NoError(t, svc.TopUpBalance(ctx, custID, 50_000_000, "idemp-list-1"))

	t.Run("list returns items and total as admin", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers?limit=50&offset=0", http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CustomerListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Total, int64(0))
		require.NotEmpty(t, resp.Items)

		var found *CustomerDTO
		for i := range resp.Items {
			if resp.Items[i].ID == custID.String() {
				found = &resp.Items[i]
				break
			}
		}
		require.NotNil(t, found, "created customer not in list")
		assert.Equal(t, "List Test Corp", found.Name)
		assert.Equal(t, "250.00", found.Balance)
	})

	t.Run("get by id returns customer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var dto CustomerDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, custID.String(), dto.ID)
		assert.Equal(t, "250.00", dto.Balance)
	})

	t.Run("get by malformed id returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers/not-a-uuid", http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get by unknown id returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers/"+uuid.New().String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("role U sees only own customer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, custID)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("role U forbidden on another customer", func(t *testing.T) {
		otherID := uuid.New()
		req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, otherID)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/customers", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCampaignsList_Handler(t *testing.T) {
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
	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Camp List Corp", 500_000_000, "USD"))

	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Test Campaign",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "idemp-camp-list-1",
	})
	require.NoError(t, err)

	t.Run("list returns items filtered by customer_id", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/campaigns?customer_id="+custID.String()+"&limit=50", http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CampaignListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Total, int64(0))

		var found bool
		for _, c := range resp.Items {
			if c.ID == campID.String() {
				found = true
				assert.Equal(t, "Test Campaign", c.Name)
				assert.Equal(t, "100.00", c.BudgetLimit)
				break
			}
		}
		assert.True(t, found, "created campaign not in list")
	})

	t.Run("role U sees only own customer campaigns", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/campaigns", http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, custID)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp CampaignListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		for _, c := range resp.Items {
			assert.Equal(t, custID.String(), c.CustomerID)
		}
	})

	t.Run("role U forbidden when customer_id is another", func(t *testing.T) {
		otherID := uuid.New()
		req, _ := http.NewRequest("GET", "/api/v1/campaigns?customer_id="+otherID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, custID)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/campaigns", http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
