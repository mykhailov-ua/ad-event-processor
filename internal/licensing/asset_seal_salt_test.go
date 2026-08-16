package licensing

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveReleaseAssetSealSalt_matchesCIFormula(t *testing.T) {
	vault := "vault-salt-example"
	tag := "v1.2.3"
	got, err := DeriveReleaseAssetSealSalt(vault, tag)
	require.NoError(t, err)
	want := sha256.Sum256([]byte(vault + ":" + tag))
	require.Equal(t, want, got)
}

func TestSealAsset_releaseSaltRoundTrip(t *testing.T) {
	salt, err := DeriveReleaseAssetSealSalt("smoke-vault", "v0.0.0-smoke")
	require.NoError(t, err)
	reset := SetAssetSealSaltForTest(salt[:])
	t.Cleanup(reset)

	var mck [32]byte
	for i := range mck {
		mck[i] = byte(i + 3)
	}
	plain := []byte("sealed with release salt")
	sealed, err := SealAsset(AssetLabelEdge, plain, mck)
	require.NoError(t, err)
	got, err := OpenAsset(AssetLabelEdge, sealed, mck)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestSealAsset_wrongReleaseSaltRejected(t *testing.T) {
	saltA, err := DeriveReleaseAssetSealSalt("vault-a", "v1.0.0")
	require.NoError(t, err)
	saltB, err := DeriveReleaseAssetSealSalt("vault-b", "v1.0.0")
	require.NoError(t, err)

	reset := SetAssetSealSaltForTest(saltA[:])
	t.Cleanup(reset)

	var mck [32]byte
	mck[0] = 0x42
	sealed, err := SealAsset(AssetLabelEdge, []byte("secret"), mck)
	require.NoError(t, err)

	resetB := SetAssetSealSaltForTest(saltB[:])
	t.Cleanup(resetB)
	_, err = OpenAsset(AssetLabelEdge, sealed, mck)
	require.Error(t, err)
}
