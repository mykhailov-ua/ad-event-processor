package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"gopkg.in/yaml.v3"
)

const (
	platformSettingsPath  = "/api/v1/settings/platform"
	platformBootstrapPath = "/api/v1/settings/platform/bootstrap"
	headerInstallToken    = "X-Install-Token"
	headerAdminAPIKey     = "X-Admin-API-Key"
)

func platformConfigJSONPath() string {
	return filepath.Join(repoRoot(), "platform_config.json")
}

func LoadPlatformConfigFromYAML(profile *InstallProfile) platformconfig.Config {
	cfg := platformconfig.Default()
	if profile.Type != "" {
		cfg.Profile = string(profile.Type)
	}
	if profile.IngressSchema != "" {
		cfg.IngressSchema = string(profile.IngressSchema)
	}
	cfg.TelemetryEnabled = profile.TelemetryEnabled
	cfg.EdgeXDP = profile.EdgeXDP
	if profile.Interface != "" {
		cfg.NetworkInterface = profile.Interface
	}
	return platformconfig.MergeDefaults(cfg)
}

func LoadPlatformConfigFromJSONFile(path string) (platformconfig.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return platformconfig.Config{}, err
	}
	return platformconfig.Parse(string(data))
}

func WritePlatformConfigJSON(path string, cfg platformconfig.Config) error {
	raw, err := platformconfig.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0644)
}

func installProfileFromConfig(cfg platformconfig.Config) InstallProfile {
	return InstallProfile{
		Type:             Profile(cfg.Profile),
		IngressSchema:    IngressSchema(cfg.IngressSchema),
		TelemetryEnabled: cfg.TelemetryEnabled,
		EdgeXDP:          cfg.EdgeXDP,
		Interface:        cfg.NetworkInterface,
	}
}

func loadConfigForApply(baseURL, adminKey string) (platformconfig.Config, InstallProfile, error) {
	if baseURL != "" && adminKey != "" {
		view, err := FetchPlatformConfigFromAPI(baseURL, adminKey)
		if err == nil {
			cfg := platformconfig.MergeDefaults(view.Config)
			return cfg, installProfileFromConfig(cfg), nil
		}
	}
	data, err := os.ReadFile("install.yaml")
	if err != nil {
		return platformconfig.Config{}, InstallProfile{}, fmt.Errorf("failed to read install.yaml: %w", err)
	}
	var profile InstallProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return platformconfig.Config{}, InstallProfile{}, fmt.Errorf("failed to parse install.yaml: %w", err)
	}
	cfg := LoadPlatformConfigFromYAML(&profile)
	jsonPath := platformConfigJSONPath()
	if jsonData, readErr := os.ReadFile(jsonPath); readErr == nil {
		if parsed, parseErr := platformconfig.Parse(string(jsonData)); parseErr == nil {
			cfg.TrackingDomain = parsed.TrackingDomain
			cfg.DefaultCurrency = parsed.DefaultCurrency
			cfg.Timezone = parsed.Timezone
			cfg.Stripe = parsed.Stripe
		}
	}
	return cfg, profile, nil
}

func FetchPlatformConfigFromAPI(baseURL, adminKey string) (platformconfig.PublicView, error) {
	url := strings.TrimRight(baseURL, "/") + platformSettingsPath
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	req.Header.Set(headerAdminAPIKey, adminKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return platformconfig.PublicView{}, fmt.Errorf("platform settings GET %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var view platformconfig.PublicView
	if err := json.Unmarshal(body, &view); err != nil {
		return platformconfig.PublicView{}, fmt.Errorf("decode platform settings: %w", err)
	}
	return view, nil
}

func BootstrapViaAPI(baseURL, token string, req platformconfig.BootstrapRequest) (platformconfig.PublicView, error) {
	url := strings.TrimRight(baseURL, "/") + platformBootstrapPath
	payload, err := json.Marshal(req)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(headerInstallToken, token)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return platformconfig.PublicView{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return platformconfig.PublicView{}, fmt.Errorf("platform bootstrap %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var view platformconfig.PublicView
	if err := json.Unmarshal(body, &view); err != nil {
		return platformconfig.PublicView{}, fmt.Errorf("decode bootstrap response: %w", err)
	}
	return view, nil
}

func writeInstallComposeEnv(cfg platformconfig.Config, dryRun bool) error {
	content := platformconfig.RenderComposeEnv(cfg)
	path := composeEnvPath()
	if dryRun {
		fmt.Printf("[Dry-Run] Would render %s (sha256=%s)\n", path, checksum(content))
		return nil
	}
	if unchanged, err := fileUnchanged(path, content); err != nil {
		return err
	} else if unchanged {
		fmt.Printf("Skipping %s (unchanged)\n", path)
		return nil
	}
	if err := writeFile(path, content, 0644); err != nil {
		return err
	}
	fmt.Printf("Rendered %s\n", path)
	return nil
}
