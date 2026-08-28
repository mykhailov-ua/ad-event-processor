package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetCustomerBalance(t *testing.T) {
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
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Balance API", 250_000_000, "USD"))

	for i := range 3 {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, $2, 'TOPUP', $3)`,
			domain.ToUUID(custID), int64((i+1)*1_000_000), fmt.Sprintf("hash-%d", i))
		require.NoError(t, err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String()+"/balance", http.NoBody)
	withSessionUser(req, tokenMaker, RoleUser, custID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var report CustomerBalanceDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&report))
	assert.Equal(t, "250.00", report.Balance)
	assert.Empty(t, report.Ledger)

	ledgerReq, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String()+"/ledger?limit=50&offset=0", http.NoBody)
	withSessionUser(ledgerReq, tokenMaker, RoleUser, custID)
	ledgerResp := httptest.NewRecorder()
	mux.ServeHTTP(ledgerResp, ledgerReq)

	require.Equal(t, http.StatusOK, ledgerResp.Code)
	var ledgerPage LedgerListResponse
	require.NoError(t, json.NewDecoder(ledgerResp.Body).Decode(&ledgerPage))
	assert.Equal(t, int64(3), ledgerPage.Total)
	assert.Len(t, ledgerPage.Items, 3)
	assert.Equal(t, "3.00", ledgerPage.Items[0].Amount)
}

func TestAPI_GetCustomerBalance_TenantIsolation(t *testing.T) {
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
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ownerID := uuid.New()
	otherID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), ownerID, "Owner", 100_000_000, "USD"))

	req, _ := http.NewRequest("GET", "/api/v1/customers/"+ownerID.String()+"/balance", http.NoBody)
	withSessionUser(req, tokenMaker, RoleUser, otherID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestAPI_ExportCustomerBalance_CSV(t *testing.T) {
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
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Export", 0, "USD"))
	_, err := pool.Exec(context.Background(), `
		INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
		VALUES ($1, 1000000, 'TOPUP', 'export-1')`, domain.ToUUID(custID))
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String()+"/balance/export?format=csv", http.NoBody)
	withSessionUser(req, tokenMaker, RoleUser, custID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/csv; charset=utf-8", resp.Header().Get("Content-Type"))
	body := resp.Body.String()
	assert.True(t, strings.HasPrefix(body, "id,customer_id"))
	assert.Contains(t, body, "TOPUP")
}

func TestAPI_ExportCustomerBalance_BufferOverflowCap(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Overflow", 0, "USD"))

	padding := strings.Repeat("X", 4096)
	for i := range 3000 {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, 1000, 'FEE', $2)`,
			domain.ToUUID(custID), fmt.Sprintf("%s-%d", padding, i))
		require.NoError(t, err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/customers/"+custID.String()+"/balance/export?format=csv", http.NoBody)
	withAdminAPIKey(req, cfg)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "true", resp.Header().Get("X-Export-Truncated"))
	assert.NotEmpty(t, resp.Header().Get("X-Next-Cursor"))

	bytesWritten, _ := strconv.Atoi(resp.Header().Get("X-Export-Bytes"))
	assert.LessOrEqual(t, bytesWritten, defaultExportChunkMaxBytes)
	assert.Greater(t, bytesWritten, defaultExportChunkMaxBytes-50_000)
}

func TestAPI_ExportCustomerBalance_RateLimit(t *testing.T) {
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	h.customerLimiter = newCustomerRateLimiterWith(0, 1)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "RL", 0, "USD"))

	url := "/api/v1/customers/" + custID.String() + "/balance/export?format=csv"
	req1, _ := http.NewRequest("GET", url, http.NoBody)
	withAdminAPIKey(req1, cfg)
	resp1 := httptest.NewRecorder()
	mux.ServeHTTP(resp1, req1)
	require.Equal(t, http.StatusOK, resp1.Code)

	req2, _ := http.NewRequest("GET", url, http.NoBody)
	withAdminAPIKey(req2, cfg)
	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, req2)
	assert.Equal(t, http.StatusTooManyRequests, resp2.Code)
}

func TestAPI_ExportCustomerBalance_CursorResume(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "Cursor", 0, "USD"))
	padding := strings.Repeat("Y", 2048)
	for i := range 6000 {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, 1000, 'FEE', $2)`,
			domain.ToUUID(custID), fmt.Sprintf("%s-%d", padding, i))
		require.NoError(t, err)
	}

	url := "/api/v1/customers/" + custID.String() + "/balance/export?format=csv"
	req1, _ := http.NewRequest("GET", url, http.NoBody)
	withAdminAPIKey(req1, cfg)
	resp1 := httptest.NewRecorder()
	mux.ServeHTTP(resp1, req1)
	require.Equal(t, http.StatusOK, resp1.Code)
	cursor := resp1.Header().Get("X-Next-Cursor")
	require.NotEmpty(t, cursor)

	req2, _ := http.NewRequest("GET", url+"&cursor="+cursor, http.NoBody)
	withAdminAPIKey(req2, cfg)
	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, req2)
	require.Equal(t, http.StatusOK, resp2.Code)
	assert.NotEmpty(t, resp2.Body.String())
}
