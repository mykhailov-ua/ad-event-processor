package verify_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/verify"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseMAC_roundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	work, err := verify.DeriveMCKWorkForRecheck(token, verify.HostFingerprint())
	require.NoError(t, err)

	fileBytes := []byte(token)
	require.NoError(t, verify.WriteLicenseMAC(path, work, fileBytes))

	stored, err := os.ReadFile(verify.LicenseMACPath(path))
	require.NoError(t, err)
	require.True(t, verify.VerifyLicenseMAC(work, fileBytes, stored))

	_, err = verify.VerifyLicenseFile(path, pub, verify.HostFingerprint(), time.Now())
	require.NoError(t, err)
}

func TestFileMAC_recheckRejectsWrongSidecar(t *testing.T) {
	verify.ResetFeatureSeedForTest()
	t.Cleanup(verify.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))
	require.NoError(t, os.WriteFile(verify.LicenseMACPath(path), make([]byte, 32), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")

	snap, err := licensing.RecheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})
	require.ErrorIs(t, err, verify.ErrLicenseMACMismatch)
	assert.Equal(t, verify.StateExpired, snap.State)
	assert.False(t, verify.FeatureSeedValid())
}

func TestFileMAC_recheckBootstrapsMissingSidecar(t *testing.T) {
	verify.ResetFeatureSeedForTest()
	t.Cleanup(verify.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")

	snap, err := licensing.RecheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})
	require.NoError(t, err)
	assert.Equal(t, verify.StateActive, snap.State)
	assert.True(t, verify.FeatureSeedValid())

	macBytes, err := os.ReadFile(verify.LicenseMACPath(path))
	require.NoError(t, err)
	assert.Len(t, macBytes, 32)
}

func TestInstallToken_writesLicenseMACInEnterpriseMode(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	require.NoError(t, verify.InstallToken(path, token, pub))

	_, err = os.Stat(verify.LicenseMACPath(path))
	require.NoError(t, err)
}
