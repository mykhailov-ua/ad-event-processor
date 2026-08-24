package installer

import (
	"fmt"
	"os"
	"strings"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/platformconfig"

	"gopkg.in/yaml.v3"
)

func RunBootstrap() error {
	root := repoRoot()
	loadDotEnv(root)
	if err := ensureEnvFile(root); err != nil {
		return err
	}

	token := strings.TrimSpace(os.Getenv("INSTALL_BOOTSTRAP_TOKEN"))
	if token == "" {
		return fmt.Errorf("INSTALL_BOOTSTRAP_TOKEN is not set in .env")
	}

	cfg, err := LoadPlatformConfigFromJSONFile(platformConfigJSONPath())
	if err != nil {
		data, readErr := os.ReadFile("install.yaml")
		if readErr != nil {
			return fmt.Errorf("platform_config.json and install.yaml missing: %w", err)
		}
		var profile InstallProfile
		if unmarshalErr := yamlUnmarshalProfile(data, &profile); unmarshalErr != nil {
			return unmarshalErr
		}
		cfg = LoadPlatformConfigFromYAML(&profile)
	}

	email := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_EMAIL"))
	password := strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_PASSWORD"))
	if email == "" || password == "" {
		return fmt.Errorf("set ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD for bootstrap")
	}

	req := platformconfig.BootstrapRequest{
		Config:        cfg,
		AdminEmail:    email,
		AdminPassword: password,
		LicenseKey:    strings.TrimSpace(config.LicenseEnv("KEY")),
		LicenseServer: strings.TrimSpace(config.LicenseEnv("SERVER")),
	}

	view, err := BootstrapViaAPI(managementBaseURL(), token, req)
	if err != nil {
		return err
	}

	fmt.Printf("Bootstrap complete\n")
	fmt.Printf("Control URL: %s\n", managementBaseURL())
	if view.ClickURLTemplate != "" {
		fmt.Printf("Click URL template: %s\n", view.ClickURLTemplate)
	}
	return nil
}

func yamlUnmarshalProfile(data []byte, profile *InstallProfile) error {
	return yaml.Unmarshal(data, profile)
}
