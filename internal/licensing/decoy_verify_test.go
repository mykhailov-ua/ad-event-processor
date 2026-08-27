package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/licensing/embedkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecoyVerify_RejectsVendorSignedJWT(t *testing.T) {
	ResetDecoyLicensedForTest()
	t.Cleanup(ResetDecoyLicensedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)

	_, err = VerifyJWT(token, pub)
	require.NoError(t, err)

	require.False(t, decoyDispatchLicenseVerify(token))
	require.False(t, DecoyLicensed())
}

func TestDecoyEmbeddedPublicKey_distinctFromProduction(t *testing.T) {
	decoyPub := decoyEmbeddedPublicKey()
	require.NotEqual(t, embedkey.EmbeddedProductionPublicKey(), decoyPub)
	require.NotContains(t, strings.ToLower(hex.EncodeToString(decoyPub)), "ede21d8e759af2ba")
}

func TestDecoyPublicKey_notVendorHex(t *testing.T) {
	raw := decoyEmbeddedPublicKey()
	require.NotContains(t, strings.ToLower(hex.EncodeToString(raw)), "ede21d8e759af2ba")
}

func TestDecoyVerify_UnreachableFromRecheck(t *testing.T) {
	decoySymbols := []string{
		"runDecoyLicenseVerify",
		"decoyDispatchLicenseVerify",
		"decoyEmbeddedPublicKey",
	}
	prodFiles := []string{
		filepath.Join("..", "ingestion", "registry_license.go"),
		"file_verify.go",
		"verify.go",
		filepath.Join("..", "..", "cmd", "tracker", "main.go"),
		filepath.Join("..", "..", "cmd", "processor", "main.go"),
	}
	for _, path := range prodFiles {
		body, err := os.ReadFile(path)
		require.NoError(t, err, path)
		for _, sym := range decoySymbols {
			require.NotContains(t, string(body), sym, "file %s", path)
		}
	}
}

func TestDecoyCold_noSnapshotEffect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(path, []byte("token"), 0o600))

	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	PublishFeatureSeed(0, false)

	require.NoError(t, DeploymentCredentialRefresh(path))
	require.NotZero(t, RuntimeEntitlementSnapshot(path))
	require.False(t, FeatureSeedValid())
}
