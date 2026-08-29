package licensing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSealAsset_roundTrip(t *testing.T) {
	var mck [32]byte
	for i := range mck {
		mck[i] = byte(i + 1)
	}
	plain := []byte("edge bpf object bytes")
	sealed, err := SealAsset(AssetLabelEdge, plain, mck)
	require.NoError(t, err)
	got, err := OpenAsset(AssetLabelEdge, sealed, mck)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestSealAsset_bitFlipRejected(t *testing.T) {
	var mck [32]byte
	for i := range mck {
		mck[i] = byte(0xAB)
	}
	sealed, err := SealAsset(AssetLabelEdge, []byte("tamper-me"), mck)
	require.NoError(t, err)
	sealed[len(sealed)-1] ^= 0xFF
	_, err = OpenAsset(AssetLabelEdge, sealed, mck)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSealTampered)
}

func TestSealAsset_wrongMCK(t *testing.T) {
	var mck [32]byte
	sealed, err := SealAsset(AssetLabelEdge, []byte("secret"), mck)
	require.NoError(t, err)
	var other [32]byte
	other[0] = 1
	_, err = OpenAsset(AssetLabelEdge, sealed, other)
	require.Error(t, err)
}
