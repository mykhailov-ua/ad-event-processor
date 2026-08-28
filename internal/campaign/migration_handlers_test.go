package campaign

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ad-event-processor/internal/migrationsource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrationTestMux() *http.ServeMux {
	h := &CampaignsHTTPHandlers{}
	mux := http.NewServeMux()
	h.registerMigrationRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	return mux
}

func TestMigrationHandlers_previewKeitaro(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)
	mux := migrationTestMux()

	body, err := json.Marshal(map[string]any{
		"source_kind": migrationsource.SourceKindKeitaroJSON,
		"payload":     json.RawMessage(raw),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var result migrationsource.PreviewResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.MappedCampaigns, 1)
	assert.Equal(t, "{{campaign.id}}", result.MappedCampaigns[0].ClickQueryParams["sub2"])
	assert.Equal(t, 1, result.SecretsStripped)
	assert.NotContains(t, strings.ToLower(rec.Body.String()), "api_token")
}

func TestMigrationHandlers_previewBinom(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "binom_facebook_campaign.json"))
	require.NoError(t, err)
	mux := migrationTestMux()

	body, err := json.Marshal(map[string]any{
		"source_kind": migrationsource.SourceKindBinomJSON,
		"payload":     json.RawMessage(raw),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var result migrationsource.PreviewResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.MappedCampaigns, 1)
	assert.Equal(t, "meta-facebook", result.MappedCampaigns[0].UITemplateID)
}

func TestMigrationHandlers_previewOversizeBody(t *testing.T) {
	mux := migrationTestMux()

	inner := `{"campaigns":[{"name":"big","tracking_url":"https://trk.example/click?x=` + strings.Repeat("a", migrationsource.MaxPayloadBytes) + `"}]}`
	body, err := json.Marshal(map[string]any{
		"source_kind": migrationsource.SourceKindKeitaroJSON,
		"payload":     json.RawMessage(inner),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMigrationHandlers_listSources(t *testing.T) {
	mux := migrationTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/migrate/sources", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp migrationsource.SourcesResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, migrationsource.MaxPayloadBytes, resp.MaxPayloadBytes)
	assert.NotEmpty(t, resp.Sources)
}
