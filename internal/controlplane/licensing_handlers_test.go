package controlplane

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
)

func TestToLicenseStatusResponse_hostIdentityWithoutWatcher(t *testing.T) {
	t.Parallel()
	resp := toLicenseStatusResponse("dep-1", "ACTIVE", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), true, nil)
	require.Equal(t, "dep-1", resp.DeploymentID)
	require.Equal(t, "ACTIVE", resp.State)
	require.Equal(t, "2027-01-01T00:00:00Z", resp.ValidUntil)
	require.NotEmpty(t, resp.HostFingerprint)
	require.NotEmpty(t, resp.HWIDv2)
	require.Nil(t, resp.HWIDMatch)
}

func TestToLicenseStatusResponse_unconfiguredIncludesHostIdentity(t *testing.T) {
	t.Parallel()
	resp := toLicenseStatusResponse("", licenseStateUnconfigured, time.Time{}, false, nil)
	require.Equal(t, licenseStateUnconfigured, resp.State)
	require.NotEmpty(t, resp.HostFingerprint)
	require.NotEmpty(t, resp.HWIDv2)
}

func TestToLicenseStatusResponse_watcherDiagnosticsEnrich(t *testing.T) {
	t.Parallel()
	match := false
	diagFn := func() (licensing.LicenseDiagnostics, bool) {
		return licensing.LicenseDiagnostics{
			DeploymentID:    "dep-watch",
			HostFingerprint: "fp-host",
			HostHWID:        "hwid-host",
			BindMode:        "hard",
			BindHWIDHash:    "hwid-expected",
			HWIDMatch:       match,
			DaysToExpiry:    12,
		}, true
	}
	resp := toLicenseStatusResponse("", licenseStateUnconfigured, time.Time{}, false, diagFn)
	require.Equal(t, "dep-watch", resp.DeploymentID)
	require.Equal(t, "fp-host", resp.HostFingerprint)
	require.Equal(t, "hwid-host", resp.HWIDv2)
	require.NotNil(t, resp.HWIDMatch)
	require.False(t, *resp.HWIDMatch)
	require.Equal(t, 12, resp.DaysToExpiry)
}
