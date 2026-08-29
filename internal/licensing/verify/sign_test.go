package verify

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	entitlements "ad-event-processor/internal/licensing/entitlements"

	"github.com/stretchr/testify/require"
)

func TestSignJWT_roundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "lic-1",
		CustomerName: "Acme",
		Plan:         entitlements.DefaultSKUCode,
		SKU:          entitlements.DefaultSKUCode,
		ValidFrom:    time.Now().UTC(),
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
		GraceDays:    7,
	}
	claims.Limits.MaxRPS = 1000

	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)

	parsed, err := VerifyJWT(token, pub)
	require.NoError(t, err)
	require.Equal(t, entitlements.DefaultSKUCode, parsed.SKU)
	require.Equal(t, uint64(1000), parsed.Limits.MaxRPS)
}

func TestSignJWT_ed25519DeterministicVector(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validUntil := validFrom.Add(72 * time.Hour)
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "vector-subject",
		DeploymentID: "dep-vector",
		CustomerName: "Vector Co",
		Plan:         entitlements.DefaultSKUCode,
		ValidFrom:    validFrom,
		ValidUntil:   validUntil,
		GraceDays:    7,
	}
	claims.Bind.Mode = "fingerprint"
	claims.Bind.Fingerprint = "fp-vector"

	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsed, err := VerifyJWT(token, pub)
	require.NoError(t, err)
	require.Equal(t, "vector-subject", parsed.Subject)
	require.Equal(t, "fp-vector", parsed.Bind.Fingerprint)
}

func TestLoadSKUFile(t *testing.T) {
	doc, err := entitlements.LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(entitlements.DefaultSKUCode)
	require.NoError(t, err)
	require.Equal(t, entitlements.DefaultSKUCode, sku.Code)
	require.True(t, sku.Features.RtbLive)
	require.True(t, sku.Features.MarginGuard)
}

func TestSKUBuildClaims(t *testing.T) {
	doc, err := entitlements.LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.DefaultSKU()
	require.NoError(t, err)
	claims := sku.BuildClaims(entitlements.IssueLicenseInput{
		CustomerName: "Buyer",
		DeploymentID: "dep-1",
		LicenseID:    "lic-1",
	})
	require.Equal(t, entitlements.DefaultSKUCode, claims.SKU)
	require.True(t, claims.Features.RtbLive)
	require.True(t, claims.Features.MultiRegion)
}

func TestGetSKU_emptyDefaultsToLicense(t *testing.T) {
	doc, err := entitlements.LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU("")
	require.NoError(t, err)
	require.Equal(t, entitlements.DefaultSKUCode, sku.Code)
}
