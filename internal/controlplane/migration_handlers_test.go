package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func TestMigrationHandlers_previewRequiresWritePermission(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: migration preview RBAC")
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

	raw, err := os.ReadFile(filepath.Join("..", "migrationsource", "testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{
		"source_kind": migrationsource.SourceKindKeitaroJSON,
		"payload":     json.RawMessage(raw),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/preview", bytes.NewReader(body))
	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), RoleSupport, uuid.New(), time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
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
