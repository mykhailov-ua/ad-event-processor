package controlplane_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign/integration"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/testutil"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
)

func TestImportAndApplyCampaignTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1,'c',0,'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1,'camp','ACTIVE',$2)`, campaignID, customerID)
	require.NoError(t, err)

	svc := controlplane.NewService(ctx, pool, nil, nil, nil)
	h := &integration.IntegrationSchemaHTTPHandlers{
		Pool:            pool,
		TemplateCatalog: svc.TemplateCatalog(pool),
		ResolveTrackingDomain: func(context.Context) string {
			return "trk.example.com"
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	importBody, _ := json.Marshal(integration.ImportTemplatesRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/templates/import", bytes.NewReader(importBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var imported []integration.IntegrationSchemaDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &imported))
	require.GreaterOrEqual(t, len(imported), 7)

	applyBody, _ := json.Marshal(integration.ApplyCampaignTemplatesRequest{
		TrafficSource:    "traffic_propellerads",
		AffiliateNetwork: "affiliate_everad",
		TrackingDomain:   "trk.example.com",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaignID.String()+"/apply-templates", bytes.NewReader(applyBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var targetURL string
	err = pool.QueryRow(ctx, `SELECT target_url FROM campaigns WHERE id = $1`, campaignID).Scan(&targetURL)
	require.NoError(t, err)
	require.Contains(t, targetURL, "sub1={sub1}")
	require.Contains(t, targetURL, "zone_id={zone_id}")

	var postbackURL string
	err = pool.QueryRow(ctx, `SELECT url_template FROM postback_configs WHERE campaign_id = $1`, campaignID).Scan(&postbackURL)
	require.NoError(t, err)
	require.Contains(t, postbackURL, "{sub1}")
	require.Contains(t, postbackURL, "{payout}")

	var statusSchemaID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT status_integration_schema_id FROM campaigns WHERE id = $1`, campaignID).Scan(&statusSchemaID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, statusSchemaID)
}
