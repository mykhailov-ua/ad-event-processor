package verify_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestArgonRecheckStretch_GoldenVector(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		MCKStretchV1 []struct {
			Name           string `json:"name"`
			MCKHex         string `json:"mck_hex"`
			DeploymentID   string `json:"deployment_id"`
			MCKWorkHex     string `json:"mck_work_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"mck_stretch_v1"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.MCKStretchV1)
	for _, fx := range doc.MCKStretchV1 {
		t.Run(fx.Name, func(t *testing.T) {
			mckBytes, err := hex.DecodeString(fx.MCKHex)
			require.NoError(t, err)
			require.Len(t, mckBytes, 32)
			var mck [32]byte
			copy(mck[:], mckBytes)

			work, err := verify.StretchMCKForRecheck(mck, fx.DeploymentID)
			require.NoError(t, err)
			require.Equal(t, fx.MCKWorkHex, hex.EncodeToString(work[:]))
			seed := verify.FeatureSeedFromMCK(work)
			require.Equal(t, fx.FeatureSeedHex, fmt.Sprintf("%08x", seed))
		})
	}
}

func TestArgonRecheckStretch_DeriveMCKWorkForRecheck(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Fixtures []struct {
			Token string `json:"token"`
			HWID  string `json:"hwid"`
		} `json:"fixtures"`
		MCKStretchV1 []struct {
			MCKWorkHex     string `json:"mck_work_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"mck_stretch_v1"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.Len(t, doc.Fixtures, 1)
	require.Len(t, doc.MCKStretchV1, 1)

	work, err := verify.DeriveMCKWorkForRecheck(doc.Fixtures[0].Token, doc.Fixtures[0].HWID)
	require.NoError(t, err)
	require.Equal(t, doc.MCKStretchV1[0].MCKWorkHex, hex.EncodeToString(work[:]))
	seed := verify.FeatureSeedFromMCK(work)
	require.Equal(t, doc.MCKStretchV1[0].FeatureSeedHex, fmt.Sprintf("%08x", seed))
}

func TestArgonRecheckStretch_UnstretchedMCKUnchanged(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Fixtures []struct {
			Token          string `json:"token"`
			HWID           string `json:"hwid"`
			MCKHex         string `json:"mck_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Fixtures)
	fx := doc.Fixtures[0]

	mck, err := verify.DeriveMCK(fx.Token, fx.HWID)
	require.NoError(t, err)
	require.Equal(t, fx.MCKHex, hex.EncodeToString(mck[:]))
	require.Equal(t, fx.FeatureSeedHex, fmt.Sprintf("%08x", verify.FeatureSeedFromMCK(mck)))

	work, err := verify.DeriveMCKWorkForRecheck(fx.Token, fx.HWID)
	require.NoError(t, err)
	require.NotEqual(t, mck, work)
}

func TestArgonRecheckStretch_DifferentDeploymentID(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	workA, err := verify.StretchMCKForRecheck(mck, "dep-a")
	require.NoError(t, err)
	workB, err := verify.StretchMCKForRecheck(mck, "dep-b")
	require.NoError(t, err)
	require.NotEqual(t, workA, workB)
}
