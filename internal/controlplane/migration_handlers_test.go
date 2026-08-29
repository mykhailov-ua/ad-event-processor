package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), ctrlhttp.RoleSupport, uuid.New(), time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
