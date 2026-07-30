package licensing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluateActivate_fingerprintRequired(t *testing.T) {
	decision := EvaluateActivate("", "lic-1", 1, nil, nil)
	require.False(t, decision.Allow)
	require.Equal(t, ErrFingerprintRequired.Error(), decision.DenyReason)
}

func TestEvaluateActivate_cloneDeniedAtLimit(t *testing.T) {
	acts := []ActivationRecord{{LicenseKey: "lic-1", Fingerprint: "fp-a"}}
	decision := EvaluateActivate("fp-b", "lic-1", 1, acts, nil)
	require.False(t, decision.Allow)
	require.Equal(t, ErrActivationLimit.Error(), decision.DenyReason)
}

func TestEvaluateActivate_sameFingerprintAllowed(t *testing.T) {
	acts := []ActivationRecord{{LicenseKey: "lic-1", Fingerprint: "fp-a"}}
	decision := EvaluateActivate("fp-a", "lic-1", 1, acts, nil)
	require.True(t, decision.Allow)
	require.False(t, decision.BindActivation)
}

func TestEvaluateActivate_deploymentFingerprintMismatch(t *testing.T) {
	dep := &DeploymentRecord{DeploymentID: "dep-1", LicenseKey: "lic-1", Fingerprint: "fp-a"}
	decision := EvaluateActivate("fp-b", "lic-1", 2, nil, dep)
	require.False(t, decision.Allow)
	require.Equal(t, ErrFingerprintMismatch.Error(), decision.DenyReason)
}

func TestEvaluateHeartbeat_bindsFirstSeen(t *testing.T) {
	dep := &DeploymentRecord{DeploymentID: "dep-1", LicenseKey: "lic-1", Fingerprint: "fp-a"}
	decision := EvaluateHeartbeat("fp-a", "lic-1", nil, dep)
	require.True(t, decision.Allow)
	require.True(t, decision.BindActivation)
}

func TestEvaluateHeartbeat_mismatchDenied(t *testing.T) {
	dep := &DeploymentRecord{DeploymentID: "dep-1", LicenseKey: "lic-1", Fingerprint: "fp-a"}
	acts := []ActivationRecord{{LicenseKey: "lic-1", Fingerprint: "fp-a"}}
	decision := EvaluateHeartbeat("fp-b", "lic-1", acts, dep)
	require.False(t, decision.Allow)
	require.Equal(t, ErrFingerprintMismatch.Error(), decision.DenyReason)
}

func TestCapHeartbeatValidUntil_capsAt72h(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	licenseUntil := now.Add(30 * 24 * time.Hour)
	capped := CapHeartbeatValidUntil(licenseUntil, now)
	require.Equal(t, now.Add(HeartbeatJWTMaxTTL), capped)
}

func TestCapHeartbeatValidUntil_respectsLicenseEnd(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	licenseUntil := now.Add(12 * time.Hour)
	capped := CapHeartbeatValidUntil(licenseUntil, now)
	require.Equal(t, licenseUntil, capped)
}
