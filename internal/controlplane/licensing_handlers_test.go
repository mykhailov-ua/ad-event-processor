package controlplane

import (
	"encoding/json"
	"testing"
	"time"

	billingdb "ad-event-processor/internal/ledger/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/trialregistry"

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

func TestEnrichLicenseStatusTrialSurface_pilotUpgrade(t *testing.T) {
	t.Setenv(trialregistry.EnvTrialSelfServeURL, "https://t.me/vendor_trial_bot")
	resp := enrichLicenseStatusTrialSurface(LicenseStatusResponse{
		State:    "ACTIVE",
		PlanCode: licensing.SKUCodePilot,
	})
	require.Equal(t, licensing.SKUCodeStarter, resp.UpgradePlanCode)
	require.Equal(t, licensing.PilotTrialValidDays, resp.PilotValidDays)
	require.Equal(t, "https://t.me/vendor_trial_bot", resp.TrialSelfServeURL)
}

func TestEnrichLicenseStatusFromRow_maxRPS(t *testing.T) {
	ent := licensing.Entitlements{Limits: licensing.Limits{MaxRPS: 5000}}
	raw, err := json.Marshal(ent)
	require.NoError(t, err)
	resp := enrichLicenseStatusFromRow(LicenseStatusResponse{State: "ACTIVE", PlanCode: "pilot"}, billingdb.BillingLicenseStatus{
		PlanCode:         "pilot",
		EntitlementsJson: raw,
	})
	require.Equal(t, uint64(5000), resp.MaxRPS)
	require.Equal(t, licensing.SKUCodeStarter, resp.UpgradePlanCode)
}
