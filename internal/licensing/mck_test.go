package licensing

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeriveMCK_GoldenVector(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		MCKInfoLabel string `json:"mck_info_label"`
		Fixtures     []struct {
			Name           string `json:"name"`
			Token          string `json:"token"`
			HWID           string `json:"hwid"`
			MCKHex         string `json:"mck_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.Equal(t, DefaultMCKInfoLabel, doc.MCKInfoLabel)
	require.Equal(t, MCKInfoLabel(), doc.MCKInfoLabel)
	require.NotEmpty(t, doc.Fixtures)
	for _, fx := range doc.Fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			mck, err := DeriveMCK(fx.Token, fx.HWID)
			require.NoError(t, err)
			require.Equal(t, fx.MCKHex, hex.EncodeToString(mck[:]))
			seed := FeatureSeedFromMCK(mck)
			require.Equal(t, fx.FeatureSeedHex, fmt.Sprintf("%08x", seed))
		})
	}
}

func TestDeriveMCK_Sensitivity(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
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

	base, err := DeriveMCK(token, "hwid-a")
	require.NoError(t, err)
	other, err := DeriveMCK(token, "hwid-b")
	require.NoError(t, err)
	require.NotEqual(t, base, other)
}

func TestDeriveMCK_JWTKeyID(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	tokenA, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)
	tokenB, err := SignJWT(claims, priv, "2026-02")
	require.NoError(t, err)

	hwid := "hwid-kid-coupling"
	mckA, err := DeriveMCK(tokenA, hwid)
	require.NoError(t, err)
	mckB, err := DeriveMCK(tokenB, hwid)
	require.NoError(t, err)
	require.NotEqual(t, mckA, mckB)
}

func TestDeriveMCK_Sensitivity100Pairs(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
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

	seen := make(map[[32]byte]struct{}, 100)
	for i := range 100 {
		hwid := fmt.Sprintf("hwid-pair-%03d", i)
		mck, err := DeriveMCK(token, hwid)
		require.NoError(t, err)
		if _, dup := seen[mck]; dup {
			t.Fatalf("duplicate MCK at pair %d", i)
		}
		seen[mck] = struct{}{}
	}
}

func TestFeatureSeed_couplingGates(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)

	ent := Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true

	require.True(t, SeedGateOpenRTB(ent))
	require.True(t, SeedGateRPS(1000))

	SetSeedCouplingRequired(true)
	require.False(t, SeedGateOpenRTB(ent))
	require.False(t, SeedGateRPS(1000))

	PublishFeatureSeed(0x1234_5678, true)
	SetMCKFeatureBitsForTest(MCKFeatureBitOpenRTB)
	require.True(t, SeedGateOpenRTB(ent))
	require.True(t, SeedGateRPS(1000))
}
