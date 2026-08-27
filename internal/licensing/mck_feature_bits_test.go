package licensing_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestMCKFeatureBits_FromStretchedGoldenVector(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		MCKStretchV1 []struct {
			MCKWorkHex     string `json:"mck_work_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"mck_stretch_v1"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.Len(t, doc.MCKStretchV1, 1)

	workBytes, err := hex.DecodeString(doc.MCKStretchV1[0].MCKWorkHex)
	require.NoError(t, err)
	var work [32]byte
	copy(work[:], workBytes)

	bits := licensing.MCKFeatureBitsFromWork(work)
	require.Equal(t, work[16], bits)

	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)

	seedBytes, err := hex.DecodeString(doc.MCKStretchV1[0].FeatureSeedHex)
	require.NoError(t, err)
	var seedU32 uint32
	for i := 0; i < 4; i++ {
		seedU32 = (seedU32 << 8) | uint32(seedBytes[i])
	}
	licensing.PublishFeatureSeed(seedU32, true)

	ent := licensing.Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true

	licensing.PublishMCKFeatureBits(bits)
	require.Equal(t, bits&licensing.MCKFeatureBitOpenRTB != 0, licensing.SeedGateOpenRTB(ent))

	licensing.PublishMCKFeatureBits(bits | licensing.MCKFeatureBitOpenRTB)
	require.True(t, licensing.SeedGateOpenRTB(ent))

	licensing.PublishMCKFeatureBits(bits &^ licensing.MCKFeatureBitOpenRTB)
	require.False(t, licensing.SeedGateOpenRTB(ent))
}
