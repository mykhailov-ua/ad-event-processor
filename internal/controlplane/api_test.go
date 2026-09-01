package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetCampaignStats_PostgresOnly(t *testing.T) {
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
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "API Stats", 500_000_000, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Stats Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "stats-camp-1",
	})
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `
		INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
		VALUES ($1, CURRENT_DATE, 100, 10, 2)`,
		domain.ToUUID(campID))
	require.NoError(t, err)

	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	url := "/api/v1/campaigns/" + campID.String() + "/stats?from=" + from + "&to=" + to + "&granularity=hour"

	req, _ := http.NewRequest("GET", url, http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, custID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var report campaign.CampaignStatsDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&report))
	assert.Equal(t, campID.String(), report.CampaignID)
	assert.Equal(t, "0.00", report.CurrentSpend)
	assert.Equal(t, int64(100), report.Metrics.Impressions)
	assert.Equal(t, int64(10), report.Metrics.Clicks)
	assert.Equal(t, int64(2), report.Metrics.Conversions)
	assert.Equal(t, "hour", report.Granularity)
	assert.Equal(t, "strong", report.Consistency)
	assert.True(t, report.Stale)
	assert.Equal(t, "pg", report.Source)
	require.Len(t, report.Hourly, 24)
	var hourlyClicks int64
	for _, bucket := range report.Hourly {
		hourlyClicks += bucket.Clicks
	}
	assert.Equal(t, int64(10), hourlyClicks)
}

func TestAPI_GetCampaignStats_TenantIsolation(t *testing.T) {
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
	require.NoError(t, svc.CreateCustomer(context.Background(), ownerID, "Owner", 500_000_000, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), campaign.CreateCampaignSpec{
		CustomerID:       ownerID,
		Name:             "Private",
		BudgetLimitMicro: 50_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "stats-camp-iso",
	})
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/v1/campaigns/"+campID.String()+"/stats", http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, otherID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestAPI_GetCampaignStats_ClickHouseStaleOK(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	conn, cleanupCH := setupClickHouseStatsTest(t)
	defer cleanupCH()
	ctx := context.Background()
	require.NoError(t, migrate.ApplyClickHouseMigrations(ctx, conn))

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
	svc.SetClickHouse(conn, database.ClickHouseQueryConfig{})
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(context.Background(), custID, "CH Stats", 500_000_000, "USD"))
	campID, err := svc.CreateCampaign(context.Background(), campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "CH Camp",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       "ASAP",
		Timezone:         "UTC",
		IdempotencyKey:   "stats-ch-1",
	})
	require.NoError(t, err)

	staleHour := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	require.NoError(t, conn.Exec(ctx, `
		INSERT INTO impressions (click_id, campaign_id, ip_address, user_agent, payload, created_at)
		VALUES (?, ?, '1.1.1.1', 'ua', '{}', ?)`,
		"stale-click-1", campID, staleHour.Add(5*time.Minute)))

	from := staleHour.Add(-time.Hour).Format(time.RFC3339)
	to := staleHour.Add(2 * time.Hour).Format(time.RFC3339)
	req, _ := http.NewRequest("GET", "/api/v1/campaigns/"+campID.String()+"/stats?from="+from+"&to="+to, http.NoBody)
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, custID)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var report campaign.CampaignStatsDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&report))
	assert.True(t, report.Stale, "ingestion lag >5m must set stale=true")
	assert.Equal(t, "eventual", report.Consistency)
	assert.Equal(t, "ch", report.Source)
	require.NotEmpty(t, report.Hourly)
}

func TestSumCampaignStatsInRange_Explain(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	campaignID := uuid.New()
	customerID := uuid.New()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'explain', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'explain', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
		VALUES ($1, CURRENT_DATE, 50, 5, 1)`,
		domain.ToUUID(campaignID))
	require.NoError(t, err)

	from := time.Now().UTC().Add(-7 * 24 * time.Hour)
	to := time.Now().UTC()
	rows, err := pool.Query(ctx, `
		EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT
			COALESCE(SUM(impressions_count), 0)::bigint,
			COALESCE(SUM(clicks_count), 0)::bigint,
			COALESCE(SUM(conversions_count), 0)::bigint
		FROM campaign_stats
		WHERE campaign_id = $1
		 AND date >= $2::date
		 AND date <= $3::date`,
		domain.ToUUID(campaignID), from, to)
	require.NoError(t, err)
	defer rows.Close()

	var plan bytes.Buffer
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	t.Logf("Postgres EXPLAIN:\n%s", plan.String())
	assert.Contains(t, plan.String(), "Index")
}
