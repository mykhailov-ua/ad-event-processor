package controlplane

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/coldpath"

	"github.com/stretchr/testify/require"
)

func TestColdPathJSON_AuthLoginRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &AuthHandler{}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := strings.Repeat("x", coldpath.DefaultMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestColdPathJSON_AlertmanagerWebhookRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &AlertmanagerWebhook{}
	mux := http.NewServeMux()
	h.Register(mux)

	body := strings.Repeat("x", coldpath.AlertmanagerWebhookMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/ops/alertmanager/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestColdPathJSON_RegionIngestRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &Handler{cfg: testRegionIngestConfig(t)}
	mux := http.NewServeMux()
	RegisterRegionIngestRoutes(mux, h)

	body := strings.Repeat("x", coldpath.RegionIngestMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/region/ingest/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-API-Key", "test-admin-key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func testRegionIngestConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.AdminAPIKey = "test-admin-key"
	cfg.MultiRegionEnabled = true
	cfg.RegionCode = 0
	return cfg
}
