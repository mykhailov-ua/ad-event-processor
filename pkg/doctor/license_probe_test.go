package doctor

import (
	"context"
	"github.com/bidshard/ad-event-processor/pkg/naming"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultProbes_includesLicenseWhenRequired(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), filepath.Join(t.TempDir(), "missing.jwt"))

	probes := DefaultProbes(ProbeDeps{Config: &config.Config{}})
	var found bool
	for _, p := range probes {
		if p.Name() == "license" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestDefaultProbes_omitsLicenseWhenDisabled(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_REQUIRED"), "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "")
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), filepath.Join(t.TempDir(), "missing.jwt"))

	probes := DefaultProbes(ProbeDeps{})
	for _, p := range probes {
		assert.NotEqual(t, "license", p.Name())
	}
}

func TestLicenseProbe_withDiagnostics(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	deps := ProbeDeps{
		LicenseState: func() (licensing.LicenseState, bool) {
			return licensing.StateActive, true
		},
		LicenseDiagnostics: func() (licensing.LicenseDiagnostics, bool) {
			return licensing.LicenseDiagnostics{
				State:            licensing.StateActive,
				DeploymentID:     "dep-cli-test",
				DaysToExpiry:     21,
				FingerprintMatch: true,
			}, true
		},
	}
	probe := licenseProbe(deps)
	result := probe.Run(context.Background())
	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Detail, "deployment_id=dep-cli-test")
	assert.Contains(t, result.Detail, "fingerprint_match=true")
}

func TestBundleLicenseInfo_fields(t *testing.T) {
	deps := ProbeDeps{
		LicenseDiagnostics: func() (licensing.LicenseDiagnostics, bool) {
			return licensing.LicenseDiagnostics{
				State:            licensing.StateActive,
				DeploymentID:     "dep-42",
				DaysToExpiry:     30,
				FingerprintMatch: true,
			}, true
		},
	}
	info := bundleLicenseInfo(deps)
	assert.Equal(t, "dep-42", info.DeploymentID)
	assert.Equal(t, 30, info.DaysToExpiry)
	assert.True(t, info.FingerprintMatch)
	assert.Equal(t, "ACTIVE", info.State)
}

func writeTempLicenseFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	if err := os.WriteFile(path, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLicenseProbeEnabled_viaFile_includesProbe(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv(naming.LegacyVendorEnvKey("PROFILE"), "")
	path := writeTempLicenseFile(t)
	t.Setenv(naming.LegacyVendorEnvKey("LICENSE_PATH"), path)

	probes := DefaultProbes(ProbeDeps{})
	found := false
	for _, p := range probes {
		if p.Name() == "license" {
			found = true
		}
	}
	assert.True(t, found)
}
