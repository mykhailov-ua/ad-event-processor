package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const productionPubKeyHexFixture = "ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70"

func TestPublicKey_EmbeddedObfuscation(t *testing.T) {
	pub := embeddedProductionPublicKey()
	require.Len(t, pub, 32)
	require.Equal(t, productionPubKeyHexFixture, hex.EncodeToString(pub))
}

func TestPublicKey_sourceNotPlainEmbeddedHex(t *testing.T) {
	for _, path := range []string{"public_key.go", "embedkey/material.go", "embedkey/xor_mask.go"} {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotContains(t, strings.ToLower(string(raw)), strings.ToLower(productionPubKeyHexFixture), path)
	}
}

func TestPublicKey_ResolveEmbeddedFallback(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_FILE", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "")
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "")
	t.Setenv("ROOT", "")
	pub, err := ResolvePublicKey()
	require.NoError(t, err)
	require.Equal(t, productionPubKeyHexFixture, hex.EncodeToString(pub))
}

func TestPublicKey_ProductionIgnoresEnvOverride(t *testing.T) {
	attackerPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	attackerHex := hex.EncodeToString(attackerPub)

	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", attackerHex)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_FILE", "")
	t.Setenv("ROOT", "")

	pub, err := ResolvePublicKey()
	require.NoError(t, err)
	require.Equal(t, productionPubKeyHexFixture, hex.EncodeToString(pub))
}

func TestPublicKey_ProductionIgnoresFileOverride(t *testing.T) {
	attackerPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	t.Chdir(dir)
	keyPath := filepath.Join(dir, defaultPublicKeyRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(keyPath), 0o755))
	require.NoError(t, os.WriteFile(keyPath, []byte(hex.EncodeToString(attackerPub)), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_FILE", "")
	t.Setenv("ROOT", dir)

	pub, err := ResolvePublicKey()
	require.NoError(t, err)
	require.Equal(t, productionPubKeyHexFixture, hex.EncodeToString(pub))
}

func TestPublicKey_ProductionOverrideAllowsEnv(t *testing.T) {
	attackerPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	attackerHex := hex.EncodeToString(attackerPub)

	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", attackerHex)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE", "1")
	t.Setenv("ROOT", "")

	pub, err := ResolvePublicKey()
	require.NoError(t, err)
	require.Equal(t, attackerHex, hex.EncodeToString(pub))
}
