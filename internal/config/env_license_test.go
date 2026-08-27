package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/pkg/naming"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLicensePathFromEnv_defaultDevPath(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", "")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), "")
	assert.Equal(t, config.DevLicenseRelPath, config.LicensePathFromEnv())
	assert.Equal(t, config.DevLicenseRelPath, config.DefaultLicensePath())
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

func TestLicenseAssetsUnsealed_devMode(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	assert.True(t, config.LicenseAssetsUnsealed())
	assert.False(t, config.LicenseSeedCouplingEnabled())
}

func TestLicenseSeedCoupling_enterpriseMode(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	assert.False(t, config.LicenseAssetsUnsealed())
	assert.True(t, config.LicenseSeedCouplingEnabled())
}

func TestLicenseSkewWatch_devModeDisabled(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	assert.False(t, config.LicenseSkewWatchEnabled())
}

func TestLicenseGuardEnv_killSwitch(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD", "0")
	assert.False(t, config.LicenseGuardEnvEnabled())
	assert.False(t, config.LicenseGuardPtraceWatchdogEnabled())
}

func TestLicenseGuardPtrace_killSwitch(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE", "0")
	assert.True(t, config.LicenseGuardEnvEnabled())
	assert.False(t, config.LicenseGuardPtraceWatchdogEnabled())
}

func TestLicensePublicKeyProductionEmbeddedOnly(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "")
	assert.True(t, config.LicensePublicKeyProductionEmbeddedOnly())

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "1")
	assert.False(t, config.LicensePublicKeyProductionEmbeddedOnly())

	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "")
	assert.False(t, config.LicensePublicKeyProductionEmbeddedOnly())
}

func TestLicenseSkewWatch_defaults(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "file")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "")
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte("token"), 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", path)
	assert.True(t, config.LicenseSkewWatchEnabled())
	assert.Equal(t, time.Hour, config.LicenseSkewWatchInterval())
	assert.Equal(t, 5*time.Minute, config.LicenseSkewWatchThreshold())
}

func TestLicenseGuardPtraceRequired_productionProfile(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE_REQUIRED", "")
	assert.True(t, config.LicenseGuardPtraceRequired())
}

func TestLicenseGuardPtraceRequired_devOverride(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE_REQUIRED", "0")
	assert.False(t, config.LicenseGuardPtraceRequired())
}

func TestLicenseGuardPtraceRequired_explicitOn(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE_REQUIRED", "1")
	assert.True(t, config.LicenseGuardPtraceRequired())
}

func TestHWIDV3Enabled_defaultOff(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "")
	assert.False(t, config.HWIDV3Enabled())
}

func TestHWIDV3Enabled_explicitOn(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "1")
	assert.True(t, config.HWIDV3Enabled())
}
