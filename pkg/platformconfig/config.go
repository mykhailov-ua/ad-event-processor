package platformconfig

import (
	"encoding/json"
	"fmt"
)

const (
	SettingsKey       = "platform_config"
	ProfileSingleVPS  = "single_vps"
	IngressESPXNative = "espx_native"
	IngressOpenRTB3   = "openrtb_3"
)

type StripeConfig struct {
	Enabled            bool   `json:"enabled"`
	SecretKey          string `json:"secret_key,omitempty"`
	WebhookSecret      string `json:"webhook_secret,omitempty"`
	CheckoutSuccessURL string `json:"checkout_success_url,omitempty"`
	CheckoutCancelURL  string `json:"checkout_cancel_url,omitempty"`
}

type Config struct {
	TrackingDomain   string       `json:"tracking_domain"`
	DefaultCurrency  string       `json:"default_currency"`
	Timezone         string       `json:"timezone"`
	IngressSchema    string       `json:"ingress_schema"`
	TelemetryEnabled bool         `json:"telemetry_enabled"`
	Stripe           StripeConfig `json:"stripe"`
	Profile          string       `json:"profile"`
	EdgeXDP          bool         `json:"edge_xdp"`
	NetworkInterface string       `json:"network_interface"`
}

type Patch struct {
	TrackingDomain   *string      `json:"tracking_domain,omitempty"`
	DefaultCurrency  *string      `json:"default_currency,omitempty"`
	Timezone         *string      `json:"timezone,omitempty"`
	IngressSchema    *string      `json:"ingress_schema,omitempty"`
	TelemetryEnabled *bool        `json:"telemetry_enabled,omitempty"`
	Stripe           *StripePatch `json:"stripe,omitempty"`
	Profile          *string      `json:"profile,omitempty"`
	EdgeXDP          *bool        `json:"edge_xdp,omitempty"`
	NetworkInterface *string      `json:"network_interface,omitempty"`
}

type StripePatch struct {
	Enabled            *bool   `json:"enabled,omitempty"`
	SecretKey          *string `json:"secret_key,omitempty"`
	WebhookSecret      *string `json:"webhook_secret,omitempty"`
	CheckoutSuccessURL *string `json:"checkout_success_url,omitempty"`
	CheckoutCancelURL  *string `json:"checkout_cancel_url,omitempty"`
}

type BootstrapRequest struct {
	Config        Config `json:"config"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	LicenseKey    string `json:"license_key,omitempty"`
	LicenseServer string `json:"license_server,omitempty"`
	DeploymentID  string `json:"deployment_id,omitempty"`
	EulaVersion   string `json:"eula_version,omitempty"`
}

func Default() Config {
	return Config{
		DefaultCurrency:  "USD",
		Timezone:         "UTC",
		IngressSchema:    IngressESPXNative,
		TelemetryEnabled: true,
		Profile:          ProfileSingleVPS,
		NetworkInterface: "eth0",
	}
}

func Parse(raw string) (Config, error) {
	if raw == "" {
		return Default(), nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse platform_config: %w", err)
	}
	cfg = MergeDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Marshal(cfg Config) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func MergeDefaults(cfg Config) Config {
	def := Default()
	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = def.DefaultCurrency
	}
	if cfg.Timezone == "" {
		cfg.Timezone = def.Timezone
	}
	if cfg.IngressSchema == "" {
		cfg.IngressSchema = def.IngressSchema
	}
	if cfg.Profile == "" {
		cfg.Profile = def.Profile
	}
	if cfg.NetworkInterface == "" {
		cfg.NetworkInterface = def.NetworkInterface
	}
	return cfg
}
