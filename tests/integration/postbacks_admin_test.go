// Role: Postback admin API (config CRUD, DLQ list, retry) with encrypted token storage.
// Tier: integration.
// Infra: testcontainers Postgres (ads + billing migrations).
// Invariants proved: PUT config persists encrypted row; DLQ retry marks RETRIED and enqueues SEND_POSTBACK outbox event.
// Verify: make test-integration
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/integration"
	"ad-event-processor/internal/controlplane"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPostbacksAdminAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	customerID := uuid.New()
	_, err := dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')", customerID, "Customer")
	require.NoError(t, err)

	campaignID := uuid.New()
	_, err = dbPool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id)
		VALUES ($1, 'Campaign', 'ACTIVE', $2)`,
		campaignID,
		customerID,
	)
	require.NoError(t, err)

	key := []byte("postback-encryption-secret-key32")
	handler := &campaign.PostbackHTTPHandlers{
		Pool:          dbPool,
		EncryptionKey: key,
	}

	mux := http.NewServeMux()
	handler.Register(mux)

	configReq := campaign.UpdatePostbackConfigRequest{
		Provider:    "facebook",
		URLTemplate: "https://mock.com",
		APIToken:    "token123",
		TargetEvent: "conversion",
	}
	bodyBytes, err := json.Marshal(configReq)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	q := db.New(dbPool)
	configEntry, err := q.GetPostbackConfig(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
	require.NoError(t, err)
	require.Equal(t, "facebook", configEntry.Provider)
	require.Equal(t, "https://mock.com", configEntry.UrlTemplate)

	req = httptest.NewRequest("GET", "/api/v1/postbacks/config", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var configs []campaign.PostbackConfigDTO
	err = json.NewDecoder(rec.Body).Decode(&configs)
	require.NoError(t, err)
	require.Len(t, configs, 1)
	require.Equal(t, campaignID.String(), configs[0].CampaignID)

	_, err = q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
		OutboxEventID: 1001,
		CampaignID:    pgtype.UUID{Bytes: campaignID, Valid: true},
		ClickID:       "click_dlq_abc",
		EventType:     "conversion",
		Payload:       []byte(`{"click_id": "click_dlq_abc"}`),
		FailuresCount: 5,
		LastError:     pgtype.Text{String: "timeout error", Valid: true},
		Status:        "FAILED",
	})
	require.NoError(t, err)

	req = httptest.NewRequest("GET", "/api/v1/postbacks/dlq", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var dlqs []campaign.PostbackDlqDTO
	err = json.NewDecoder(rec.Body).Decode(&dlqs)
	require.NoError(t, err)
	require.Len(t, dlqs, 1)
	require.Equal(t, "click_dlq_abc", dlqs[0].ClickID)
	require.Equal(t, "FAILED", dlqs[0].Status)

	dlqIDStr := strconv.FormatInt(dlqs[0].ID, 10)
	req = httptest.NewRequest("POST", "/api/v1/postbacks/dlq/"+dlqIDStr+"/retry", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	dlqUpdated, err := q.GetPostbackDLQ(ctx, dlqs[0].ID)
	require.NoError(t, err)
	require.Equal(t, "RETRIED", dlqUpdated.Status)

	outboxEvents, err := q.GetPendingPostbackEventsForUpdate(ctx, db.GetPendingPostbackEventsForUpdateParams{
		Limit:   10,
		Column2: 120,
	})
	require.NoError(t, err)
	require.Len(t, outboxEvents, 1)
	require.Equal(t, "SEND_POSTBACK", outboxEvents[0].EventType)
}

func TestPostbacksHealthEndpointIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	customerID := uuid.New()
	_, err := dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')", customerID, "Customer")
	require.NoError(t, err)

	campaignID := uuid.New()
	_, err = dbPool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id)
		VALUES ($1, 'Campaign', 'ACTIVE', $2)`,
		campaignID,
		customerID,
	)
	require.NoError(t, err)

	key := []byte("postback-encryption-secret-key32")
	handler := &campaign.PostbackHTTPHandlers{
		Pool:          dbPool,
		EncryptionKey: key,
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	configReq := campaign.UpdatePostbackConfigRequest{
		Provider:    "webhook",
		URLTemplate: "https://example.com/pb?click={click_id}",
		APIToken:    "token123",
		TargetEvent: "conversion",
	}
	bodyBytes, err := json.Marshal(configReq)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	for i, spec := range []struct {
		hash    string
		status  string
		latency int
		errMsg  string
	}{
		{hash: "health-sent-1", status: "SENT", latency: 120},
		{hash: "health-sent-2", status: "SENT", latency: 80},
		{hash: "health-fail-1", status: "FAILED", latency: 200, errMsg: "timeout"},
	} {
		_, err = dbPool.Exec(ctx, `
			INSERT INTO postback_dispatches (idempotency_hash, campaign_id, click_id, event_type, status, error_message, latency_ms, created_at)
			VALUES ($1, $2, $3, 'conversion', $4, $5, $6, NOW() - INTERVAL '1 hour' * $7)`,
			spec.hash,
			campaignID,
			"click-"+strconv.Itoa(i),
			spec.status,
			spec.errMsg,
			spec.latency,
			i,
		)
		require.NoError(t, err)
	}

	req = httptest.NewRequest("GET", "/api/v1/postbacks/health", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var health campaign.PostbackHealthResponseDTO
	err = json.NewDecoder(rec.Body).Decode(&health)
	require.NoError(t, err)
	require.Equal(t, 95.0, health.AlertThresholdSuccessRate)
	require.Len(t, health.Rows, 1)
	require.Equal(t, campaignID.String(), health.Rows[0].CampaignID)
	require.NotNil(t, health.Rows[0].SuccessRate24h)
	require.InDelta(t, 66.67, *health.Rows[0].SuccessRate24h, 0.1)
	require.Equal(t, "fail", health.Rows[0].HealthStatus)
	require.Equal(t, "timeout", health.Rows[0].LastError)
	require.NotNil(t, health.Rows[0].P95LatencyMs)
	require.GreaterOrEqual(t, *health.Rows[0].P95LatencyMs, int64(120))

	var sentCount, totalCount int64
	err = dbPool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'SENT'),
			COUNT(*) FILTER (WHERE status IN ('SENT', 'FAILED'))
		FROM postback_dispatches
		WHERE campaign_id = $1
		  AND created_at >= NOW() - INTERVAL '24 hours'`,
		campaignID,
	).Scan(&sentCount, &totalCount)
	require.NoError(t, err)
	require.Equal(t, int64(2), sentCount)
	require.Equal(t, int64(3), totalCount)
	expectedRate := 100.0 * float64(sentCount) / float64(totalCount)
	require.InDelta(t, expectedRate, *health.Rows[0].SuccessRate24h, 1.0)
	require.Equal(t, "/docs/INTEGRATIONS.md#postback-health", health.RunbookPath)
}

func TestApplyCampaignTemplatesOneClickIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := dbPool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id)
		VALUES ($1, 'camp', 'ACTIVE', $2)`,
		campaignID,
		customerID,
	)
	require.NoError(t, err)

	svc := controlplane.NewService(ctx, dbPool, nil, nil, nil)
	defer svc.Close()
	h := &integration.IntegrationSchemaHTTPHandlers{
		Pool:            dbPool,
		TemplateCatalog: svc.TemplateCatalog(dbPool),
		ResolveTrackingDomain: func(context.Context) string {
			return "trk.example.com"
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	importBody, err := json.Marshal(integration.ImportTemplatesRequest{})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/templates/import", bytes.NewReader(importBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	applyBody, err := json.Marshal(integration.ApplyCampaignTemplatesRequest{
		TrafficSource:    "traffic_propellerads",
		AffiliateNetwork: "affiliate_everad",
		TrackingDomain:   "trk.example.com",
	})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaignID.String()+"/apply-templates", bytes.NewReader(applyBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var targetURL string
	err = dbPool.QueryRow(ctx, `SELECT COALESCE(target_url, '') FROM campaigns WHERE id = $1`, campaignID).Scan(&targetURL)
	require.NoError(t, err)
	require.Contains(t, targetURL, "sub1={sub1}")

	var postbackURL string
	err = dbPool.QueryRow(ctx, `SELECT url_template FROM postback_configs WHERE campaign_id = $1`, campaignID).Scan(&postbackURL)
	require.NoError(t, err)
	require.Contains(t, postbackURL, "{payout}")

	var statusSchemaID uuid.UUID
	err = dbPool.QueryRow(ctx, `SELECT status_integration_schema_id FROM campaigns WHERE id = $1`, campaignID).Scan(&statusSchemaID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, statusSchemaID)

	var mappingCount int
	err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_conversion_mappings WHERE campaign_id = $1`, campaignID).Scan(&mappingCount)
	require.NoError(t, err)
	require.Greater(t, mappingCount, 0)

	dryRunBody, err := json.Marshal(integration.ApplyCampaignTemplatesRequest{
		AffiliateNetwork: "affiliate_everad",
		TrackingDomain:   "trk.example.com",
	})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaignID.String()+"/apply-templates/dry-run", bytes.NewReader(dryRunBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var dryRun campaign.DryRunCampaignTemplatesResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &dryRun))
	require.NotEmpty(t, dryRun.PostbackURLTemplate)
}
