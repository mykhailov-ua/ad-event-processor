package verify_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	entitlements "ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensing/verify"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func signClockAnchorTestLicense(t *testing.T, priv ed25519.PrivateKey, validUntil time.Time) string {
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().UTC().Add(-time.Hour),
		ValidUntil:   validUntil,
		GraceDays:    7,
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)
	return token
}

func TestClockAnchor_bootstrapThenRewind_holdout(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	token := signClockAnchorTestLicense(t, priv, time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))

	hostFP := verify.HostFingerprint()
	require.NoError(t, verify.VerifyOrBootstrapLicenseMAC(licensePath, pub, hostFP))

	now := time.Unix(1_700_000_000, 0).UTC()
	require.NoError(t, verify.UpdateClockAnchor(licensePath, pub, hostFP, now, time.Minute))
	require.NoError(t, verify.CheckClockAnchor(licensePath, pub, hostFP, now.Add(time.Hour), time.Minute))

	err = verify.CheckClockAnchor(licensePath, pub, hostFP, now.Add(-2*time.Hour), time.Minute)
	require.ErrorIs(t, err, verify.ErrClockAnchorRewind)
}

func TestClockAnchor_tamperMAC_holdout(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	token := signClockAnchorTestLicense(t, priv, time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))

	hostFP := verify.HostFingerprint()
	require.NoError(t, verify.VerifyOrBootstrapLicenseMAC(licensePath, pub, hostFP))

	now := time.Now().UTC()
	require.NoError(t, verify.UpdateClockAnchor(licensePath, pub, hostFP, now, time.Minute))

	raw, err := os.ReadFile(verify.ClockAnchorPath(licensePath))
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff
	require.NoError(t, os.WriteFile(verify.ClockAnchorPath(licensePath), raw, 0o600))

	err = verify.CheckClockAnchor(licensePath, pub, hostFP, now, time.Minute)
	require.ErrorIs(t, err, verify.ErrClockAnchorTamper)
}
