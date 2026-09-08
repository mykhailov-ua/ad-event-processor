package verify_test

import (
	"testing"
	"time"

	entitlements "ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/assert"
)

func TestClaimsRevoked_inJWT(t *testing.T) {
	claims := &entitlements.LicenseClaims{
		ValidFrom:  time.Now().Add(-time.Hour),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Revoked:    true,
	}
	state := entitlements.DetermineState(claims, time.Now(), false)
	assert.Equal(t, entitlements.StateRevoked, state)
}

func TestBuildLicenseDiagnostics_fingerprintMismatch(t *testing.T) {
	claims := &entitlements.LicenseClaims{
		DeploymentID: "dep-1",
		ValidUntil:   time.Now().Add(48 * time.Hour),
	}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "expected-fp-not-host"

	diag := verify.BuildLicenseDiagnostics(claims, entitlements.StateActive, time.Now())
	assert.Equal(t, "dep-1", diag.DeploymentID)
	assert.False(t, diag.FingerprintMatch)
	assert.Greater(t, diag.DaysToExpiry, 0)
}
