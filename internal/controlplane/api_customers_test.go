package controlplane

import (
	"context"
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

func TestManagementAPI_Customers(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}

	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	err := svc.CreateCustomer(ctx, custID, "Acme Corp", 150_500_000, "USD")
	require.NoError(t, err)

	err = svc.TopUpBalance(ctx, custID, 50_000_000, "idemp-hash-1")
	require.NoError(t, err)

	t.Run("ListCustomers", func(t *testing.T) {
		customers, total, err := svc.ListCustomers(ctx, 10, 0)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		require.NotEmpty(t, customers)

		var found *CustomerDTO
		for i := range customers {
			if customers[i].ID == custID.String() {
				found = &customers[i]
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "Acme Corp", found.Name)
		assert.Equal(t, "200.50", found.Balance)
	})

	t.Run("GetCustomerByID", func(t *testing.T) {
		cust, err := svc.GetCustomerDTO(ctx, custID)
		require.NoError(t, err)
		assert.Equal(t, custID.String(), cust.ID)
		assert.Equal(t, "200.50", cust.Balance)
	})

	t.Run("GetCustomerLedger", func(t *testing.T) {
		ledger, total, err := svc.ListCustomerLedger(ctx, custID, 50, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, ledger, 1)
		assert.Equal(t, "50.00", ledger[0].Amount)
		assert.Equal(t, "TOPUP", ledger[0].Type)
	})

	t.Run("CustomerIsolation_Forbidden", func(t *testing.T) {
		otherCustID := uuid.New()

		req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String()+"/balance", http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, otherCustID)

		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
	})
}
