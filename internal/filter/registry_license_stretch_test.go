package filter

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
	"github.com/stretchr/testify/require"
)

func TestRegistry_recheckUsesStretchedMCKSeed(t *testing.T) {
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
		Limits:       licensing.Limits{MaxRPS: 1000},
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	registry := NewRegistry(nil)
	registry.recheckLicenseFile(context.Background(), licensing.FileLicenseRecheckConfig{
		Path:   path,
		PubKey: pub,
	})

	wantSeed, err := licensing.FeatureSeedFromLicenseFileRecheck(path, pub, licensing.HostFingerprint())
	require.NoError(t, err)

	gotSeed, ok := registry.GetLicenseFeatureSeed()
	require.True(t, ok)
	require.Equal(t, wantSeed, gotSeed)

	mck, err := licensing.DeriveMCKFromLicenseFile(path, pub, licensing.HostFingerprint())
	require.NoError(t, err)
	require.NotEqual(t, wantSeed, licensing.FeatureSeedFromMCK(mck))
}
