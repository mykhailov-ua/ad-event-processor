package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceBilling_costCenterPatchAndExport(t *testing.T) {
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

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Agency West", 0, "USD"))

	patchBody := `{"cost_center":"agency-west"}`
	patchReq, _ := http.NewRequest("PATCH", "/api/v1/customers/"+custID.String()+"/cost-center", strings.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	withAdminAPIKey(patchReq, cfg)
	patchResp := httptest.NewRecorder()
	mux.ServeHTTP(patchResp, patchReq)
	require.Equal(t, http.StatusOK, patchResp.Code)
	assert.Contains(t, patchResp.Body.String(), `"cost_center":"agency-west"`)

	usageDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `
		INSERT INTO billing.usage_daily (customer_id, usage_date, meter, value)
		VALUES ($1, $2, 'accepted_events', 42)`, custID, usageDate)
	require.NoError(t, err)

	exportURL := "/api/v1/billing/usage/export?format=csv&from=2026-08-01&to=2026-08-31&customer_id=" + custID.String()
	exportReq, _ := http.NewRequest("GET", exportURL, http.NoBody)
	withAdminAPIKey(exportReq, cfg)
	exportResp := httptest.NewRecorder()
	mux.ServeHTTP(exportResp, exportReq)
	require.Equal(t, http.StatusOK, exportResp.Code)
	body := exportResp.Body.String()
	assert.True(t, strings.HasPrefix(body, "customer_id,customer_name,cost_center"))
	assert.Contains(t, body, "agency-west")
	assert.Contains(t, body, "accepted_events")
	assert.Contains(t, body, "42")
}

func TestWorkspaceBilling_usageExportTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
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

	ctx := context.Background()
	victimID := uuid.New()
	attackerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, victimID, "Victim", 0, "USD"))
	require.NoError(t, svc.CreateCustomer(ctx, attackerID, "Attacker", 0, "USD"))

	usageDate := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `
		INSERT INTO billing.usage_daily (customer_id, usage_date, meter, value)
		VALUES ($1, $2, 'events', 999)`, domain.ToUUID(victimID), usageDate)
	require.NoError(t, err)

	path := "/api/v1/billing/usage/export?format=csv&from=2026-08-01&to=2026-08-31&customer_id=" + victimID.String()
	req, _ := http.NewRequest("GET", path, http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, attackerID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	require.Equal(t, http.StatusForbidden, resp.Code)
	assert.NotContains(t, resp.Body.String(), "999")
}

func TestParseUsageExportCursor_roundTrip(t *testing.T) {
	t.Parallel()
	customerID := uuid.New()
	cursor, err := billingadmin.ParseUsageExportCursor(customerID.String() + "|2026-08-01|events")
	require.NoError(t, err)
	require.True(t, cursor.Valid)
	assert.Equal(t, customerID, cursor.CustomerID)
	assert.Equal(t, "events", cursor.Meter)
	assert.Equal(t, customerID.String()+"|2026-08-01|events", cursor.Encode())
}

func TestNormalizeCostCenter_rejectsLongValue(t *testing.T) {
	t.Parallel()
	_, err := billingadmin.NormalizeCostCenter(strings.Repeat("x", 65))
	require.Error(t, err)
}
