package licenseissue

import (
	"os"
	"strings"
)

const (
	EnvSKUFile        = "LICENSE_VENDOR_SKU_FILE"
	EnvPrivateKeyFile = "LICENSE_VENDOR_PRIVATE_KEY_FILE"
	EnvKeyID          = "LICENSE_VENDOR_KEY_ID"
	EnvTrialRegistry  = "VENDOR_TRIAL_REGISTRY"
)

func ConfigFromEnv() Config {
	cfg := Config{
		SKUFile:        "deploy/vendor/sku.yaml",
		PrivateKeyFile: "deploy/vendor/license_private.key",
		KeyID:          strings.TrimSpace(os.Getenv(EnvKeyID)),
	}
	if path := strings.TrimSpace(os.Getenv(EnvSKUFile)); path != "" {
		cfg.SKUFile = path
	}
	if path := strings.TrimSpace(os.Getenv(EnvPrivateKeyFile)); path != "" {
		cfg.PrivateKeyFile = path
	}
	if path := strings.TrimSpace(os.Getenv(EnvTrialRegistry)); path != "" {
		cfg.TrialRegistry = path
	}
	return cfg
}
