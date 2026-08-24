package licensing_test

import (
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
)

func TestClaimsRevoked_inJWT(t *testing.T) {
	claims := &licensing.LicenseClaims{
		ValidFrom:  time.Now().Add(-time.Hour),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Revoked:    true,
	}
	state := licensing.DetermineState(claims, time.Now(), false)
	assert.Equal(t, licensing.StateRevoked, state)
}

func TestBuildLicenseDiagnostics_fingerprintMismatch(t *testing.T) {
	claims := &licensing.LicenseClaims{
		DeploymentID: "dep-1",
		ValidUntil:   time.Now().Add(48 * time.Hour),
	}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "expected-fp-not-host"

	diag := licensing.BuildLicenseDiagnostics(claims, licensing.StateActive, time.Now())
	assert.Equal(t, "dep-1", diag.DeploymentID)
	assert.False(t, diag.FingerprintMatch)
	assert.Greater(t, diag.DaysToExpiry, 0)
}
