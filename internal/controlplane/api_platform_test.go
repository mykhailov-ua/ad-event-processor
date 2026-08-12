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

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type platformAuthStub struct {
	registerCalled bool
}

func (s *platformAuthStub) Register(_ context.Context, _, email, password, role, customerID string) error {
	s.registerCalled = true
	if email == "" || password == "" || role != "A" || customerID != "" {
		return errValidation("unexpected register args")
	}
	return nil
}

func TestManagementAPI_PlatformSettingsCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	installRoot := t.TempDir()
	cfg := &config.Config{
		AdminAPIKey:           "test-secret",
		InstallBootstrapToken: "install-token",
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	authStub := &platformAuthStub{}
	h := NewHandler(svc, cfg, nil, nil, nil, nil)
	mux := http.NewServeMux()
	registry := h.BuildAdminAPIRegistry(pool, []redis.UniversalClient{rdb})
	registry.PlatformHTTP.AuthClient = authStub
	registry.PlatformHTTP.Cfg = cfg
	adminapi.RegisterRoutes(mux, registry)

	t.Run("GET before bootstrap", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/platform", nil)
		withAdminAPIKey(req, cfg)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var view platformconfig.PublicView
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
		assert.False(t, view.BootstrapComplete)
		assert.Equal(t, platformconfig.Default().DefaultCurrency, view.Config.DefaultCurrency)
	})

	t.Run("bootstrap", func(t *testing.T) {
		body, err := json.Marshal(platformconfig.BootstrapRequest{
			Config: platformconfig.Config{
				TrackingDomain:  "trk.example.com",
				DefaultCurrency: "USD",
			},
			AdminEmail:    "admin@example.com",
			AdminPassword: "secret-pass",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/platform/bootstrap", bytes.NewReader(body))
		req.Header.Set("X-Install-Token", "install-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, authStub.registerCalled)
		var view platformconfig.PublicView
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
		assert.True(t, view.BootstrapComplete)
		assert.Equal(t, "trk.example.com", view.Config.TrackingDomain)
	})

	t.Run("PATCH restart required", func(t *testing.T) {
		patch := platformconfig.Patch{
			IngressSchema: strPtr(platformconfig.IngressOpenRTB3),
		}
		body, err := json.Marshal(patch)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/platform", bytes.NewReader(body))
		withAdminAPIKey(req, cfg)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var view platformconfig.PublicView
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
		assert.Contains(t, view.RestartRequired, "ingress_schema")
		assert.Equal(t, platformconfig.IngressOpenRTB3, view.Config.IngressSchema)
	})

	t.Run("apply compose env", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{"install_root": installRoot})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/platform/apply", bytes.NewReader(body))
		withAdminAPIKey(req, cfg)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			WrittenPath string `json:"written_path"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		expected := filepath.Join(installRoot, "install.compose.env")
		assert.Equal(t, expected, resp.WrittenPath)
		raw, err := os.ReadFile(expected)
		require.NoError(t, err)
		assert.Contains(t, string(raw), "TRACKER_INGRESS_SCHEMA="+platformconfig.IngressOpenRTB3)
	})

	t.Run("GET after apply clears pending restart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/platform", nil)
		withAdminAPIKey(req, cfg)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var view platformconfig.PublicView
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
		assert.True(t, view.BootstrapComplete)
		assert.Empty(t, view.RestartRequired)
	})
}

func strPtr(s string) *string { return &s }
