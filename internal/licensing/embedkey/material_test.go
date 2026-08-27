package embedkey

import (
	"encoding/hex"
	"testing"
)

const productionPubKeyHexFixture = "ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70"

func TestEmbeddedProductionPublicKey_matchesFixture(t *testing.T) {
	pub := EmbeddedProductionPublicKey()
	if got := hex.EncodeToString(pub); got != productionPubKeyHexFixture {
		t.Fatalf("pub hex = %q want %q", got, productionPubKeyHexFixture)
	}
}
