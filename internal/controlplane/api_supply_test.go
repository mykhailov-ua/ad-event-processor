package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupplyAPI_CRUDAndExport(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	exportDir := t.TempDir()
	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	cfg.Management.SupplyExportPath = exportDir

	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)

	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO system_settings (key, value) VALUES
			('supply_owner_domain', 'owner.example.com'),
			('supply_manager_domain', 'manager.example.com')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	require.NoError(t, err)

	seller, err := svc.CreateSeller(ctx, SellerCreateSpec{
		SellerID:   "pub-001",
		Domain:     "publisher.example.com",
		SellerType: "PUBLISHER",
		Name:       "Example Publisher",
	})
	require.NoError(t, err)
	assert.Equal(t, "pub-001", seller.SellerID)

	_, err = svc.CreateAdsTxtEntry(ctx, AdsTxtEntryCreateSpec{
		Domain:             "google.com",
		PublisherAccountID: "pub-12345",
		Relationship:       "RESELLER",
		CertAuthorityID:    "f08c47fec0942fa0",
	})
	require.NoError(t, err)

	worker := NewOutboxWorker(svc)
	n, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 2, n, "seller + ads-txt each enqueue one outbox event")

	sellersPath := filepath.Join(exportDir, "sellers.json")
	adsPath := filepath.Join(exportDir, "ads.txt")
	require.FileExists(t, sellersPath)
	require.FileExists(t, adsPath)

	sellersRaw, err := os.ReadFile(sellersPath)
	require.NoError(t, err)
	var sellersDoc map[string]any
	require.NoError(t, json.Unmarshal(sellersRaw, &sellersDoc))
	assert.Equal(t, "1.0", sellersDoc["version"])
	sellersArr := sellersDoc["sellers"].([]any)
	require.Len(t, sellersArr, 1)

	adsRaw, err := os.ReadFile(adsPath)
	require.NoError(t, err)
	adsText := string(adsRaw)
	assert.Contains(t, adsText, "OWNERDOMAIN=owner.example.com")
	assert.Contains(t, adsText, "MANAGERDOMAIN=manager.example.com")
	assert.Contains(t, adsText, "google.com, pub-12345, RESELLER, f08c47fec0942fa0")

	sellersJSON, err := svc.GetSellersJSON(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, sellersJSON)

	builtAds, err := svc.BuildAdsTxt(ctx)
	require.NoError(t, err)
	assert.Contains(t, builtAds, "OWNERDOMAIN=owner.example.com")

	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Supply Co", 200_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		Name:             "Chain Camp",
		BudgetLimitMicro: 10_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "supply-chain-test",
	})
	require.NoError(t, err)

	_, err = svc.UpdateCampaignSupplyChain(ctx, campID, []SupplyChainNode{
		{ASI: "exchange.example.com", SID: "1234", HP: 1},
	})
	require.NoError(t, err)

	var auditCount int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM admin_audit_log WHERE action = 'UPDATE_CAMPAIGN_SUPPLY_CHAIN' AND target_id = $1`,
		campID).Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), auditCount)
}

func TestSupplyAPI_RBAC(t *testing.T) {
	if testing.Short() {
		t.Skip()
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
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	managerID := uuid.New()
	attachManager := func(req *http.Request) {
		token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), RoleManager, managerID, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	}

	body, _ := json.Marshal(map[string]string{"ip": "1.2.3.4", "source": "manual"})
	req, _ := http.NewRequest("POST", "/api/v1/ops/blacklist", bytes.NewReader(body))
	attachManager(req)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}
