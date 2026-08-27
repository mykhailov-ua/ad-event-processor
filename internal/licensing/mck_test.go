package licensing_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

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
	require.Equal(t, licensing.DefaultMCKInfoLabel, doc.MCKInfoLabel)
	require.Equal(t, licensing.MCKInfoLabel(), doc.MCKInfoLabel)
	require.NotEmpty(t, doc.Fixtures)
	for _, fx := range doc.Fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			mck, err := licensing.DeriveMCK(fx.Token, fx.HWID)
			require.NoError(t, err)
			require.Equal(t, fx.MCKHex, hex.EncodeToString(mck[:]))
			seed := licensing.FeatureSeedFromMCK(mck)
			require.Equal(t, fx.FeatureSeedHex, fmt.Sprintf("%08x", seed))
		})
	}
}

func TestDeriveMCK_Sensitivity(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	_, err = licensing.VerifyJWT(token, pub)
	require.NoError(t, err)

	base, err := licensing.DeriveMCK(token, "hwid-a")
	require.NoError(t, err)
	other, err := licensing.DeriveMCK(token, "hwid-b")
	require.NoError(t, err)
	require.NotEqual(t, base, other)
}

func TestDeriveMCK_JWTKeyID(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	tokenA, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	tokenB, err := licensing.SignJWT(claims, priv, "2026-02")
	require.NoError(t, err)

	hwid := "hwid-kid-coupling"
	mckA, err := licensing.DeriveMCK(tokenA, hwid)
	require.NoError(t, err)
	mckB, err := licensing.DeriveMCK(tokenB, hwid)
	require.NoError(t, err)
	require.NotEqual(t, mckA, mckB)
}

func TestDeriveMCK_Sensitivity100Pairs(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	_, err = licensing.VerifyJWT(token, pub)
	require.NoError(t, err)

	seen := make(map[[32]byte]struct{}, 100)
	for i := 0; i < 100; i++ {
		hwid := fmt.Sprintf("hwid-pair-%03d", i)
		mck, err := licensing.DeriveMCK(token, hwid)
		require.NoError(t, err)
		if _, dup := seen[mck]; dup {
			t.Fatalf("duplicate MCK at pair %d", i)
		}
		seen[mck] = struct{}{}
	}
}

func TestFeatureSeed_couplingGates(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)

	ent := licensing.Entitlements{}
	ent.Features.OpenRTBEngine = true
	ent.Features.RtbLive = true

	require.True(t, licensing.SeedGateOpenRTB(ent))
	require.True(t, licensing.SeedGateRPS(1000))

	licensing.SetSeedCouplingRequired(true)
	require.False(t, licensing.SeedGateOpenRTB(ent))
	require.False(t, licensing.SeedGateRPS(1000))

	licensing.PublishFeatureSeed(0x1234_5678, true)
	licensing.SetMCKFeatureBitsForTest(licensing.MCKFeatureBitOpenRTB)
	require.True(t, licensing.SeedGateOpenRTB(ent))
	require.True(t, licensing.SeedGateRPS(1000))
}
