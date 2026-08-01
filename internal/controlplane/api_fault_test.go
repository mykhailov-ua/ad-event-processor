package controlplane

import (
	"espx/pkg/faultproof"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/clickhouse/migrate"
	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	apiFaultWorkers       = 24
	apiFaultVictimBalance = int64(1_234_567_890)
)

func TestFault_APITenantIsolation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("fault integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	cfg.Management.RateLimitRPS = 100_000
	cfg.Management.RateLimitBurst = 10_000
	authMW, tokenMaker := integrationTestAuth(t, rdb, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	h.customerLimiter = newCustomerRateLimiter()
	h.customerLimiter.limit = 1000
	h.customerLimiter.burst = 128
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	victimID := uuid.New()
	attackerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, victimID, "Victim", apiFaultVictimBalance, "USD"))
	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       victimID,
		Name:             "Secret",
		BudgetLimitMicro: 99_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "fault-tenant-camp",
	})
	require.NoError(t, err)

	ownerReq, _ := http.NewRequest("GET", "/api/v1/customers/"+victimID.String()+"/balance", nil)
	withSessionUser(ownerReq, tokenMaker, RoleUser, victimID)
	ownerResp := httptest.NewRecorder()
	mux.ServeHTTP(ownerResp, ownerReq)
	require.Equal(t, http.StatusOK, ownerResp.Code)
	var ownerReport CustomerBalanceDTO
	require.NoError(t, json.NewDecoder(ownerResp.Body).Decode(&ownerReport))
	leakMarker := ownerReport.Balance
	require.NotEmpty(t, leakMarker)
	paths := []string{
		"/api/v1/customers/" + victimID.String() + "/balance",
		"/api/v1/campaigns/" + campID.String() + "/stats",
		"/api/v1/customers/" + victimID.String() + "/balance/export?format=csv",
	}

	var forbidden atomic.Int32
	var wg sync.WaitGroup
	wg.Add(apiFaultWorkers * len(paths))
	for i := 0; i < apiFaultWorkers; i++ {
		for _, path := range paths {
			go func(p string) {
				defer wg.Done()
				req, _ := http.NewRequest("GET", p, nil)
				withSessionUser(req, tokenMaker, RoleUser, attackerID)
				resp := httptest.NewRecorder()
				mux.ServeHTTP(resp, req)
				if resp.Code == http.StatusForbidden {
					forbidden.Add(1)
				}
				assert.Equal(t, http.StatusForbidden, resp.Code)
				assert.NotContains(t, resp.Body.String(), leakMarker, "cross-tenant body leak on %s", p)
			}(path)
		}
	}
	wg.Wait()

	require.Equal(t, int32(apiFaultWorkers*len(paths)), forbidden.Load())

	faultproof.Log(t, "api_tenant_isolation", map[string]string{
		"subsystem":     "management_api",
		"workers":       strconv.Itoa(apiFaultWorkers),
		"endpoints":     strconv.Itoa(len(paths)),
		"forbidden":     strconv.Itoa(int(forbidden.Load())),
		"leak_detected": "false",
		"baseline_ok":   "true",
		"fault_type":    "concurrent_cross_tenant_probe",
	})
}

func TestFault_APIChLagStaleOK(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("fault integration test")
	}

	conn, chContainer, cleanupCH := setupClickHouseStatsContainer(t)
	defer cleanupCH()
	ctx := context.Background()
	require.NoError(t, migrate.ApplyClickHouseMigrations(ctx, conn))

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	cfg.Management.RateLimitRPS = 100_000
	cfg.Management.RateLimitBurst = 10_000
	authMW, tokenMaker := integrationTestAuth(t, rdb, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	svc.SetClickHouse(conn, database.CHQueryConfig{})
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "CH lag", 500_000_000, "USD"))
	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Lag Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "fault-ch-lag",
	})
	require.NoError(t, err)

	const pgImpressions int64 = 77
	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
		VALUES ($1, CURRENT_DATE, $2, 3, 1)`,
		domain.ToUUID(campID), pgImpressions)
	require.NoError(t, err)

	now := time.Now().UTC()
	queryHour := now.Truncate(time.Hour)
	from := queryHour.Add(-2 * time.Hour).Format(time.RFC3339)
	to := queryHour.Add(2 * time.Hour).Format(time.RFC3339)
	statsURL := "/api/v1/campaigns/" + campID.String() + "/stats?from=" + from + "&to=" + to

	staleHour := queryHour.Add(-2 * time.Hour)
	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO impressions (click_id, campaign_id, ip_address, user_agent, payload, created_at)
		VALUES (?, ?, '1.1.1.1', 'ua', '{}', ?)`,
		"stale-click", campID, staleHour.Add(5*time.Minute)))

	reqStale, _ := http.NewRequest("GET", statsURL, nil)
	withSessionUser(reqStale, tokenMaker, RoleUser, custID)
	respStale := httptest.NewRecorder()
	mux.ServeHTTP(respStale, reqStale)
	require.Equal(t, http.StatusOK, respStale.Code)

	var staleReport CampaignStatsDTO
	require.NoError(t, json.NewDecoder(respStale.Body).Decode(&staleReport))
	require.True(t, staleReport.Stale, "ingestion lag >5m must set stale=true")
	assert.Equal(t, "eventual", staleReport.Consistency)
	assert.Equal(t, pgImpressions, staleReport.Metrics.Impressions, "PG metrics must remain readable during CH lag")

	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO impressions (click_id, campaign_id, ip_address, user_agent, payload, created_at)
		VALUES (?, ?, '1.1.1.1', 'ua', '{}', ?)`,
		"fresh-click", campID, now))

	reqFresh, _ := http.NewRequest("GET", statsURL, nil)
	withSessionUser(reqFresh, tokenMaker, RoleUser, custID)
	respFresh := httptest.NewRecorder()
	mux.ServeHTTP(respFresh, reqFresh)
	require.Equal(t, http.StatusOK, respFresh.Code)
	var freshReport CampaignStatsDTO
	require.NoError(t, json.NewDecoder(respFresh.Body).Decode(&freshReport))
	require.False(t, freshReport.Stale, "fresh CH ingest must clear stale flag")

	stopFaultContainer(t, chContainer)
	requireFaultActive(t, func() bool {
		return conn.Ping(ctx) != nil
	}, "clickhouse must be unreachable after container stop")
	startFaultContainer(t, chContainer)

	faultproof.Log(t, "api_ch_lag_stale_ok", map[string]string{
		"subsystem":      "management_api",
		"campaign_id":    campID.String(),
		"stale":          "true",
		"http_status":    "200",
		"pg_impressions": strconv.FormatInt(pgImpressions, 10),
		"baseline_ok":    "true",
		"fault_type":     "clickhouse_ingestion_lag",
		"fault_verify":   "ch_container_stopped",
		"ch_lag_minutes": "120",
	})
}

func TestFault_LedgerExportCursor(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("fault integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	cfg.Management.RateLimitRPS = 100_000
	cfg.Management.RateLimitBurst = 10_000
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	h.customerLimiter = newCustomerRateLimiter()
	h.customerLimiter.limit = 1000
	h.customerLimiter.burst = 64
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Export fault", 0, "USD"))

	padding := strings.Repeat("X", 4096)
	const rows = 3000
	for i := 0; i < rows; i++ {
		_, err := pool.Exec(ctx, `
			INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
			VALUES ($1, 1000, 'FEE', $2)`,
			domain.ToUUID(custID), fmt.Sprintf("%s-%d", padding, i))
		require.NoError(t, err)
	}

	exportURL := "/api/v1/customers/" + custID.String() + "/balance/export?format=csv"

	var wg sync.WaitGroup
	var truncations atomic.Int32
	wg.Add(apiFaultWorkers)
	for i := 0; i < apiFaultWorkers; i++ {
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", exportURL, nil)
			withAdminAPIKey(req, cfg)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)
			if resp.Code == http.StatusOK && resp.Header().Get("X-Export-Truncated") == "true" {
				truncations.Add(1)
				bytesWritten, _ := strconv.Atoi(resp.Header().Get("X-Export-Bytes"))
				assert.LessOrEqual(t, bytesWritten, ledgerExportMaxBytes)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(apiFaultWorkers), truncations.Load())

	req1, _ := http.NewRequest("GET", exportURL, nil)
	withAdminAPIKey(req1, cfg)
	resp1 := httptest.NewRecorder()
	mux.ServeHTTP(resp1, req1)
	require.Equal(t, http.StatusOK, resp1.Code)
	require.Equal(t, "true", resp1.Header().Get("X-Export-Truncated"))
	cursor := resp1.Header().Get("X-Next-Cursor")
	require.NotEmpty(t, cursor)

	page1IDs := parseExportCSVIds(t, resp1.Body.String())
	bytesWritten, _ := strconv.Atoi(resp1.Header().Get("X-Export-Bytes"))
	require.LessOrEqual(t, bytesWritten, ledgerExportMaxBytes)
	require.Greater(t, bytesWritten, ledgerExportMaxBytes-50_000)

	req2, _ := http.NewRequest("GET", exportURL+"&cursor="+cursor, nil)
	withAdminAPIKey(req2, cfg)
	resp2 := httptest.NewRecorder()
	mux.ServeHTTP(resp2, req2)
	require.Equal(t, http.StatusOK, resp2.Code)
	page2IDs := parseExportCSVIds(t, resp2.Body.String())
	require.NotEmpty(t, page2IDs)

	cursorID, err := strconv.ParseInt(cursor, 10, 64)
	require.NoError(t, err)
	for _, id := range page2IDs {
		rowID, err := strconv.ParseInt(id, 10, 64)
		require.NoError(t, err)
		require.Less(t, rowID, cursorID, "cursor page must only return older rows")
	}
	require.NotEmpty(t, page1IDs)
	maxPage1, err := strconv.ParseInt(page1IDs[0], 10, 64)
	require.NoError(t, err)
	require.GreaterOrEqual(t, maxPage1, cursorID)

	faultproof.Log(t, "ledger_export_cursor", map[string]string{
		"subsystem":        "management_api",
		"customer_id":      custID.String(),
		"rows_seeded":      strconv.Itoa(rows),
		"workers":          strconv.Itoa(apiFaultWorkers),
		"truncated":        "true",
		"max_bytes":        strconv.Itoa(ledgerExportMaxBytes),
		"cursor_resume_ok": "true",
		"baseline_ok":      "true",
		"fault_type":       "oversized_ledger_export",
	})
}

func parseExportCSVIds(t *testing.T, body string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	if len(lines) <= 1 {
		return nil
	}
	ids := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		comma := strings.IndexByte(line, ',')
		if comma <= 0 {
			continue
		}
		ids = append(ids, line[:comma])
	}
	return ids
}
