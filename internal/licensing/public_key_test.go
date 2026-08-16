package licensing

import (
	"encoding/hex"
	"os"
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
	raw, err := os.ReadFile("public_key.go")
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(raw)), strings.ToLower(productionPubKeyHexFixture))
}

func TestPublicKey_ResolveEmbeddedFallback(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", "")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_FILE", "")
	t.Setenv("ROOT", "")
	pub, err := ResolvePublicKey()
	require.NoError(t, err)
	require.Equal(t, productionPubKeyHexFixture, hex.EncodeToString(pub))
}
