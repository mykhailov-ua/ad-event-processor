package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/naming"
)

func RunLicense(cmd string) error {
	switch cmd {
	case "install":
		return installLicenseFromEnv()
	case "activate":
		return activateLicense()
	case "status":
		return licenseStatus()
	case "host-id":
		return printHostLicenseIdentity()
	default:
		return fmt.Errorf("unknown license command: %s", cmd)
	}
}

func printHostLicenseIdentity() error {
	fp := licensing.HostFingerprint()
	hwid := licensing.HostHWID()
	fmt.Printf("host_fingerprint=%s\n", fp)
	fmt.Printf("hwid_v2=%s\n", hwid)
	return nil
}

func installLicenseFromEnv() error {
	src := os.Getenv(naming.LegacyVendorEnvKey("LICENSE_SRC"))
	if src == "" {
		return fmt.Errorf("set %s to the license JWT file path", naming.LegacyVendorEnvKey("LICENSE_SRC"))
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(licensePath()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(licensePath(), data, 0o600); err != nil {
		return err
	}

	fmt.Printf("License installed to %s\n", licensePath())
	return nil
}

func activateLicense() error {
	serverURL := os.Getenv(naming.LegacyVendorEnvKey("LICENSE_SERVER"))
	licenseKey := os.Getenv(naming.LegacyVendorEnvKey("LICENSE_KEY"))
	deploymentID := os.Getenv(naming.LegacyVendorEnvKey("DEPLOYMENT_ID"))
	fingerprint := os.Getenv(naming.LegacyVendorEnvKey("DEPLOYMENT_FINGERPRINT"))

	if serverURL == "" || licenseKey == "" || deploymentID == "" {
		return fmt.Errorf("set %s, %s, and %s",
			naming.LegacyVendorEnvKey("LICENSE_SERVER"),
			naming.LegacyVendorEnvKey("LICENSE_KEY"),
			naming.LegacyVendorEnvKey("DEPLOYMENT_ID"))
	}

	client := licensing.NewLicenseClient(serverURL, licenseKey, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.Activate(ctx, deploymentID, fingerprint)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(licensePath()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(licensePath(), []byte(token), 0o600); err != nil {
		return err
	}

	fmt.Printf("License activated and saved to %s\n", licensePath())
	return nil
}

func licenseStatus() error {
	data, err := os.ReadFile(licensePath())
	if err != nil {
		return fmt.Errorf("read license: %w", err)
	}

	claims, err := licensing.DecodeUnverified(string(data))
	if err != nil {
		return err
	}

	state := licensing.DetermineState(claims, time.Now(), false)
	fmt.Printf("License status: %s (deployment_id=%s valid_until=%s)\n",
		state, claims.DeploymentID, claims.ValidUntil.Format(time.RFC3339))
	return nil
}
