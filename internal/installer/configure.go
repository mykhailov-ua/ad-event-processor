package installer

import (
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/branding"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"gopkg.in/yaml.v3"
)

func RunConfigure(interactive bool) error {
	cfg := platformconfig.Default()
	cfg.Profile = platformconfig.ProfileSingleVPS
	cfg.IngressSchema = platformconfig.IngressAdEventProcessorNative

	if interactive {
		fmt.Println(branding.ProductName() + " setup (single_vps)")

		fmt.Print("Tracking domain (e.g. trk.example.com): ")
		var trackingDomain string
		_, _ = fmt.Scanln(&trackingDomain)
		cfg.TrackingDomain = strings.TrimSpace(trackingDomain)

		fmt.Print("Default currency [USD]: ")
		var currency string
		_, _ = fmt.Scanln(&currency)
		if strings.TrimSpace(currency) != "" {
			cfg.DefaultCurrency = strings.ToUpper(strings.TrimSpace(currency))
		}

		fmt.Print("Timezone [UTC]: ")
		var timezone string
		_, _ = fmt.Scanln(&timezone)
		if strings.TrimSpace(timezone) != "" {
			cfg.Timezone = strings.TrimSpace(timezone)
		}

		fmt.Print("Enable telemetry? (Y/n): ")
		var telemetry string
		_, _ = fmt.Scanln(&telemetry)
		if strings.EqualFold(strings.TrimSpace(telemetry), "n") {
			cfg.TelemetryEnabled = false
		}

		fmt.Print("Enable Stripe payments? (y/N): ")
		var stripeYN string
		_, _ = fmt.Scanln(&stripeYN)
		if strings.EqualFold(strings.TrimSpace(stripeYN), "y") {
			cfg.Stripe.Enabled = true
			fmt.Print("Stripe secret key: ")
			var secretKey string
			_, _ = fmt.Scanln(&secretKey)
			cfg.Stripe.SecretKey = strings.TrimSpace(secretKey)
			fmt.Print("Stripe webhook secret: ")
			var webhookSecret string
			_, _ = fmt.Scanln(&webhookSecret)
			cfg.Stripe.WebhookSecret = strings.TrimSpace(webhookSecret)
		}
	}

	cfg = platformconfig.MergeDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	profile := installProfileFromConfig(cfg)
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	yamlData, err := yaml.Marshal(&profile)
	if err != nil {
		return err
	}
	if err := os.WriteFile("install.yaml", yamlData, 0644); err != nil {
		return err
	}

	if err := WritePlatformConfigJSON(platformConfigJSONPath(), cfg); err != nil {
		return err
	}

	fmt.Println("Configuration saved to install.yaml and platform_config.json")
	return nil
}
