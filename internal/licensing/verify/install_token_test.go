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

func TestInstallToken_writesVerifiedJWT(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := verify.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
	}
	token, err := verify.SignJWT(claims, priv, verify.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, verify.InstallToken(path, token, pub))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, token, string(data))

	verified, err := verify.VerifyJWT(token, pub)
	require.NoError(t, err)
	assert.Equal(t, claims.DeploymentID, verified.DeploymentID)
}
