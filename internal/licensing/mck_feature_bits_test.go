package licensing

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

	bits := MCKFeatureBitsFromWork(work)
	require.Equal(t, work[16], bits)

	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	SetSeedCouplingRequired(true)

	seedBytes, err := hex.DecodeString(doc.MCKStretchV1[0].FeatureSeedHex)
	require.NoError(t, err)
	var seedU32 uint32
	for i := range 4 {
		seedU32 = (seedU32 << 8) | uint32(seedBytes[i])
	}
	PublishFeatureSeed(seedU32, true)

	ent := Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true

	PublishMCKFeatureBits(bits)
	require.Equal(t, bits&MCKFeatureBitOpenRTB != 0, SeedGateOpenRTB(ent))

	PublishMCKFeatureBits(bits | MCKFeatureBitOpenRTB)
	require.True(t, SeedGateOpenRTB(ent))

	PublishMCKFeatureBits(bits &^ MCKFeatureBitOpenRTB)
	require.False(t, SeedGateOpenRTB(ent))
}
