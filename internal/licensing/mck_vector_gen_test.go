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
	"ad-event-processor/internal/licensing/entitlements"

	"github.com/stretchr/testify/require"
)

func TestGenMCKVectorArtifacts(t *testing.T) {
	if os.Getenv("WRITE_MCK_VECTORS") == "" {
		t.Skip("set WRITE_MCK_VECTORS=1 to regenerate mck_derivation.json")
	}
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "mck-vector-subject",
		DeploymentID: "dep-mck-vector",
		CustomerName: "Vector Co",
		Plan:         entitlements.DefaultSKUCode,
		ValidFrom:    validFrom,
		ValidUntil:   validFrom.Add(72 * time.Hour),
		GraceDays:    7,
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	tel := licensing.HWIDTelemetry{
		DMIUUID:  "mck-fixture-dmi",
		DiskID:   "mck-fixture-disk",
		MAC:      "aa:bb:cc:dd:ee:ff",
		CPUModel: "MCK Fixture CPU",
		CPUCores: 4,
	}
	hwid := licensing.HashHWIDFromTelemetry(tel)
	mck, err := licensing.DeriveMCK(token, hwid)
	require.NoError(t, err)
	seedU32 := licensing.FeatureSeedFromMCK(mck)
	out := struct {
		MCKInfoLabel string `json:"mck_info_label"`
		Fixtures     []struct {
			Name           string `json:"name"`
			Token          string `json:"token"`
			HWID           string `json:"hwid"`
			MCKHex         string `json:"mck_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"fixtures"`
		MCKStretchV1 []struct {
			Name           string `json:"name"`
			MCKHex         string `json:"mck_hex"`
			DeploymentID   string `json:"deployment_id"`
			MCKWorkHex     string `json:"mck_work_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		} `json:"mck_stretch_v1"`
	}{
		MCKInfoLabel: licensing.MCKInfoLabel(),
		Fixtures: []struct {
			Name           string `json:"name"`
			Token          string `json:"token"`
			HWID           string `json:"hwid"`
			MCKHex         string `json:"mck_hex"`
			FeatureSeedHex string `json:"feature_seed_hex"`
		}{{
			Name:           "deterministic_fixture",
			Token:          token,
			HWID:           hwid,
			MCKHex:         hex.EncodeToString(mck[:]),
			FeatureSeedHex: fmt.Sprintf("%08x", seedU32),
		}},
	}
	mckWork, err := licensing.StretchMCKForRecheck(mck, claims.DeploymentID)
	require.NoError(t, err)
	out.MCKStretchV1 = []struct {
		Name           string `json:"name"`
		MCKHex         string `json:"mck_hex"`
		DeploymentID   string `json:"deployment_id"`
		MCKWorkHex     string `json:"mck_work_hex"`
		FeatureSeedHex string `json:"feature_seed_hex"`
	}{{
		Name:           "deterministic_fixture",
		MCKHex:         hex.EncodeToString(mck[:]),
		DeploymentID:   claims.DeploymentID,
		MCKWorkHex:     hex.EncodeToString(mckWork[:]),
		FeatureSeedHex: fmt.Sprintf("%08x", licensing.FeatureSeedFromMCK(mckWork)),
	}}
	raw, err := json.MarshalIndent(out, "", " ")
	require.NoError(t, err)
	path := filepath.Join("testdata", "mck_derivation.json")
	require.NoError(t, os.WriteFile(path, append(raw, '\n'), 0o644))
}
