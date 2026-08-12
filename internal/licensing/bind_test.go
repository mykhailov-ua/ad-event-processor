package licensing_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyDeploymentBind_hardMismatch(t *testing.T) {
	claims := &licensing.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.Bind.Fingerprint = "expected-fp"
	err := licensing.VerifyDeploymentBind(claims, "other-fp")
	require.ErrorIs(t, err, licensing.ErrFingerprintMismatch)
}

func TestVerifyDeploymentBind_softAllowsMismatch(t *testing.T) {
	claims := &licensing.LicenseClaims{}
	claims.Bind.Mode = "soft"
	claims.Bind.Fingerprint = "expected-fp"
	require.NoError(t, licensing.VerifyDeploymentBind(claims, "other-fp"))
}

func TestVerifyLicenseFile_validAndExpired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-48 * time.Hour),
		ValidUntil:   time.Now().Add(-24 * time.Hour),
		GraceDays:    0,
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	verified, err := licensing.VerifyLicenseFile(path, pub, "", time.Now())
	require.NoError(t, err)
	assert.Equal(t, licensing.StateExpired, verified.State)

	claims.ValidUntil = time.Now().Add(24 * time.Hour)
	token, err = licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	verified, err = licensing.VerifyLicenseFile(path, pub, "", time.Now())
	require.NoError(t, err)
	assert.Equal(t, licensing.StateActive, verified.State)
}
