package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignJWT_roundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	claims := LicenseClaims{
		Issuer:       "espx-license",
		Subject:      "lic-1",
		CustomerName: "Acme",
		Plan:         "ingest_pro",
		SKU:          "ingest_pro",
		ValidFrom:    time.Now().UTC(),
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
		GraceDays:    7,
	}
	claims.Limits.MaxRPS = 1000

	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)

	parsed, err := VerifyJWT(token, pub)
	require.NoError(t, err)
	require.Equal(t, "ingest_pro", parsed.SKU)
	require.Equal(t, uint64(1000), parsed.Limits.MaxRPS)
}

func TestLoadSKUFile(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU("ingest_pro")
	require.NoError(t, err)
	require.Equal(t, "Ingest + Antifraud Pro", sku.DisplayName)
}

func TestSKUBuildClaims(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU("network_enterprise")
	require.NoError(t, err)
	claims := sku.BuildClaims(IssueLicenseInput{
		CustomerName: "Buyer",
		DeploymentID: "dep-1",
		LicenseID:    "lic-1",
	})
	require.Equal(t, "network_enterprise", claims.SKU)
	require.True(t, claims.Features.RtbLive)
}
