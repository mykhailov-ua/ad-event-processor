package licensing_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"espx/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolvePublicKeyForKID_cohortFile(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	kid := "cohort-test"
	keyDir := filepath.Join(dir, "deploy", "vendor", "keys", kid)
	require.NoError(t, os.MkdirAll(keyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(keyDir, "license_public.key"), []byte(hexEncodePub(pub)), 0o644))

	t.Setenv("ROOT", dir)

	resolved, err := licensing.ResolvePublicKeyForKID(kid)
	require.NoError(t, err)
	require.Equal(t, ed25519.PublicKey(pub), resolved)

	claims := licensing.LicenseClaims{
		Issuer:       "espx-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, kid)
	require.NoError(t, err)

	verified, err := licensing.VerifyJWTResolved(token)
	require.NoError(t, err)
	require.Equal(t, claims.DeploymentID, verified.DeploymentID)
}

func hexEncodePub(pub ed25519.PublicKey) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(pub)*2)
	for i, b := range pub {
		out[i*2] = hexdigits[b>>4]
		out[i*2+1] = hexdigits[b&0x0f]
	}
	return string(out)
}
