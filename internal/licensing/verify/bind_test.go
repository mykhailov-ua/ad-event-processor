package verify_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing/verify"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyDeploymentBind_hardMismatch(t *testing.T) {
	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "expected-fp"
	err := verify.VerifyDeploymentBind(claims, "other-fp")
	require.ErrorIs(t, err, verify.ErrFingerprintMismatch)
}

func TestVerifyDeploymentBind_legacyFingerprintMatch(t *testing.T) {
	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "legacy-fp"
	require.NoError(t, verify.VerifyDeploymentBind(claims, "legacy-fp"))
}

func TestVerifyDeploymentBind_hwidTakesPrecedenceOverFingerprint(t *testing.T) {
	tel := verify.HWIDTelemetry{
		DMIUUID:  "precedence-dmi",
		DiskID:   "precedence-disk",
		MAC:      "11:22:33:44:55:66",
		CPUModel: "Precedence CPU",
		CPUCores: 4,
	}
	expected := verify.HashHWIDFromTelemetry(tel)
	restore := verify.SetHWIDCollectForTest(func() verify.HWIDTelemetry { return tel })
	defer restore()

	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "wrong-legacy-fp"
	claims.HWIDHash = expected
	require.NoError(t, verify.VerifyDeploymentBind(claims, "wrong-legacy-fp"))
}

func TestVerifyDeploymentBind_softAllowsMismatch(t *testing.T) {
	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "soft"
	claims.Bind.Fingerprint = "expected-fp"
	require.NoError(t, verify.VerifyDeploymentBind(claims, "other-fp"))
}

func TestVerifyLicenseFile_validAndExpired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-48 * time.Hour),
		ValidUntil:   time.Now().Add(-24 * time.Hour),
		GraceDays:    0,
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	verified, err := verify.VerifyLicenseFile(path, pub, "", time.Now())
	require.NoError(t, err)
	assert.Equal(t, verify.StateExpired, verified.State)

	claims.ValidUntil = time.Now().Add(24 * time.Hour)
	token, err = verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	verified, err = verify.VerifyLicenseFile(path, pub, "", time.Now())
	require.NoError(t, err)
	assert.Equal(t, verify.StateActive, verified.State)
}
