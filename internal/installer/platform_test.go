package installer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/pkg/platformconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPlatformConfigFromYAML(t *testing.T) {
	profile := &InstallProfile{
		Type:             ProfileSingleVPS,
		IngressSchema:    IngressSchemaAdEventProcessorNative,
		TelemetryEnabled: true,
		Interface:        "eth0",
	}
	cfg := LoadPlatformConfigFromYAML(profile)
	assert.Equal(t, platformconfig.ProfileSingleVPS, cfg.Profile)
	assert.Equal(t, platformconfig.IngressAdEventProcessorNative, cfg.IngressSchema)
	assert.True(t, cfg.TelemetryEnabled)
}

func TestFetchPlatformConfigFromAPI(t *testing.T) {
	cfg := platformconfig.Default()
	cfg.TrackingDomain = "trk.example.com"
	view := platformconfig.Public(cfg, true, nil)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, platformSettingsPath, r.URL.Path)
		assert.Equal(t, "secret-key", r.Header.Get(headerAdminAPIKey))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	}))
	defer srv.Close()

	got, err := FetchPlatformConfigFromAPI(srv.URL, "secret-key")
	require.NoError(t, err)
	assert.Equal(t, "trk.example.com", got.Config.TrackingDomain)
	assert.True(t, got.BootstrapComplete)
}

func TestBootstrapViaAPI(t *testing.T) {
	cfg := platformconfig.Default()
	cfg.TrackingDomain = "trk.example.com"
	req := platformconfig.BootstrapRequest{
		Config:        cfg,
		AdminEmail:    "admin@example.com",
		AdminPassword: "secret-pass",
	}
	view := platformconfig.Public(cfg, true, nil)
	view.ClickURLTemplate = platformconfig.ClickURLTemplate(cfg.TrackingDomain)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, platformBootstrapPath, r.URL.Path)
		assert.Equal(t, "install-token", r.Header.Get(headerInstallToken))
		var body platformconfig.BootstrapRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "admin@example.com", body.AdminEmail)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	}))
	defer srv.Close()

	got, err := BootstrapViaAPI(srv.URL, "install-token", req)
	require.NoError(t, err)
	assert.True(t, got.BootstrapComplete)
	assert.Contains(t, got.ClickURLTemplate, "trk.example.com")
}

func TestInstallProfileFromConfig(t *testing.T) {
	cfg := platformconfig.Default()
	cfg.Profile = platformconfig.ProfileSingleVPS
	cfg.IngressSchema = platformconfig.IngressAdEventProcessorNative
	profile := installProfileFromConfig(cfg)
	assert.Equal(t, ProfileSingleVPS, profile.Type)
	assert.Equal(t, IngressSchemaAdEventProcessorNative, profile.IngressSchema)
}
