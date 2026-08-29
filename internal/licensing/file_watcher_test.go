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
	"ad-event-processor/internal/licensing/entitlements"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFileLicenseRecheck_publishesStretchedSeed(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := entitlements.LicenseClaims{
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
	require.Equal(t, entitlements.StateActive, snap.State)
	require.True(t, snap.SeedValid)
	require.True(t, entitlements.FeatureSeedValid())
	require.Equal(t, snap.FeatureSeed, entitlements.FeatureSeed())

	want, err := licensing.FeatureSeedFromLicenseFileRecheck(path, pub, licensing.HostFingerprint())
	require.NoError(t, err)
	require.Equal(t, want, snap.FeatureSeed)
}

func TestStartFileLicenseRecheck_background(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := entitlements.LicenseClaims{
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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	licensing.StartFileLicenseRecheck(ctx, licensing.FileLicenseRecheckConfig{
		Path:     path,
		PubKey:   pub,
		Interval: time.Hour,
	})
	require.True(t, entitlements.FeatureSeedValid())
	cancel()
	licensing.WaitFileLicenseRecheckForTest()
}
