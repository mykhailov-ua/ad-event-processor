package licensing_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseMAC_roundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	work, err := licensing.DeriveMCKWorkForRecheck(token, licensing.HostFingerprint())
	require.NoError(t, err)

	fileBytes := []byte(token)
	require.NoError(t, licensing.WriteLicenseMAC(path, work, fileBytes))

	stored, err := os.ReadFile(licensing.LicenseMACPath(path))
	require.NoError(t, err)
	require.True(t, licensing.VerifyLicenseMAC(work, fileBytes, stored))

	_, err = licensing.VerifyLicenseFile(path, pub, licensing.HostFingerprint(), time.Now())
	require.NoError(t, err)
}

func TestFileMAC_recheckRejectsWrongSidecar(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))
	require.NoError(t, os.WriteFile(licensing.LicenseMACPath(path), make([]byte, 32), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")

	snap, err := licensing.RecheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})
	require.ErrorIs(t, err, licensing.ErrLicenseMACMismatch)
	assert.Equal(t, licensing.StateExpired, snap.State)
	assert.False(t, licensing.FeatureSeedValid())
}

func TestFileMAC_recheckBootstrapsMissingSidecar(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")

	snap, err := licensing.RecheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})
	require.NoError(t, err)
	assert.Equal(t, licensing.StateActive, snap.State)
	assert.True(t, licensing.FeatureSeedValid())

	macBytes, err := os.ReadFile(licensing.LicenseMACPath(path))
	require.NoError(t, err)
	assert.Len(t, macBytes, 32)
}

func TestInstallToken_writesLicenseMACInEnterpriseMode(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	require.NoError(t, licensing.InstallToken(path, token, pub))

	_, err = os.Stat(licensing.LicenseMACPath(path))
	require.NoError(t, err)
}
