package trialregistry

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultRegistryPath   = "deploy/vendor/trial_registry.json"
	EnvRegistryPath       = "BIDSHARD_VENDOR_TRIAL_REGISTRY"
	EnvHWIDCooldownDays   = "BIDSHARD_VENDOR_TRIAL_HWID_COOLDOWN_DAYS"
	EnvForceEnabled        = "BIDSHARD_VENDOR_TRIAL_FORCE"
	EnvVendorTrialBotToken = "BIDSHARD_VENDOR_TRIAL_BOT_TOKEN"
	defaultHWIDCooldownDays = 60
)

type Config struct {
	RegistryPath    string
	HWIDCooldown    time.Duration
	ForceEnvAllowed bool
}

func ConfigFromEnv() Config {
	cfg := Config{
		RegistryPath: DefaultRegistryPath,
		HWIDCooldown: defaultHWIDCooldownDays * 24 * time.Hour,
	}
	if path := strings.TrimSpace(os.Getenv(EnvRegistryPath)); path != "" {
		cfg.RegistryPath = path
	}
	if raw := strings.TrimSpace(os.Getenv(EnvHWIDCooldownDays)); raw != "" {
		if days, err := strconv.Atoi(raw); err == nil && days > 0 {
			cfg.HWIDCooldown = time.Duration(days) * 24 * time.Hour
		}
	}
	cfg.ForceEnvAllowed = strings.TrimSpace(os.Getenv(EnvForceEnabled)) == "1"
	return cfg
}

func ForceOverrideAllowed() bool {
	return strings.TrimSpace(os.Getenv(EnvForceEnabled)) == "1"
}
