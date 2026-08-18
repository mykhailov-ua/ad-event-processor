package controlplane_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane"
	"github.com/bidshard/ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSchemaHandlers_createAndApply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
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

	h := &controlplane.IntegrationSchemaHTTPHandlers{Pool: pool}
	mux := http.NewServeMux()
	h.Register(mux)

	schemaBody := []byte(`{
		"version": 1,
		"url_template": "https://aff.example.com/pb?click_id={click_id}&sub1={sub1}",
		"placeholders": ["click_id", "sub1"]
	}`)
	createReq := map[string]any{
		"name":    "m6-outbound-smoke",
		"version": 1,
		"schema":  json.RawMessage(schemaBody),
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/schemas", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created controlplane.IntegrationSchemaDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Equal(t, "outbound_postback", created.Kind)

	applyBody, _ := json.Marshal(map[string]string{"campaign_id": campaignID.String()})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/integration/schemas/"+created.ID+"/apply", bytes.NewReader(applyBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var cfgURL string
	err = pool.QueryRow(ctx, `SELECT url_template FROM postback_configs WHERE campaign_id = $1`, campaignID).Scan(&cfgURL)
	require.NoError(t, err)
	require.Contains(t, cfgURL, "{click_id}")

	var linked uuid.UUID
	err = pool.QueryRow(ctx, `SELECT integration_schema_id FROM campaigns WHERE id = $1`, campaignID).Scan(&linked)
	require.NoError(t, err)
	require.Equal(t, created.ID, linked.String())
}

func TestIntegrationSchemaHandlers_invalidSchema400(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	h := &controlplane.IntegrationSchemaHTTPHandlers{Pool: pool}
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(map[string]any{
		"name":    "bad",
		"version": 1,
		"schema":  json.RawMessage(`{"version":1,"foo":"bar"}`),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integration/schemas", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
