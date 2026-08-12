package config_test

import (
	"github.com/bidshard/ad-event-processor/pkg/naming"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
)

func TestLicenseRequiredFromEnv_defaultOffInDev(t *testing.T) {
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "")
	assert.False(t, config.LicenseRequiredFromEnv())
}

func TestLicenseRequiredFromEnv_productionProfile(t *testing.T) {
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "production")
	assert.True(t, config.LicenseRequiredFromEnv())
}

func TestLicenseRequiredFromEnv_explicit(t *testing.T) {
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "1")
	assert.True(t, config.LicenseRequiredFromEnv())
}

func TestLicenseEnv_prefersADEventProcessor(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "0")
	assert.Equal(t, "1", config.LicenseEnv("REQUIRED"))
}

func TestLicenseProbeEnabled_withFile(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "")
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte("not-empty"), 0o600))
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), path)
	assert.True(t, config.LicenseFilePresent())
	assert.True(t, config.LicenseProbeEnabled())
}

func TestLicenseProbeEnabled_requiredOnly(t *testing.T) {
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), filepath.Join(t.TempDir(), "missing.jwt"))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	assert.False(t, config.LicenseFilePresent())
	assert.True(t, config.LicenseProbeEnabled())
}
