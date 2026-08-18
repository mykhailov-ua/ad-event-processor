package edge

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEdgeSealed_MCKMatchesLicenseFilePath(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := licensing.HWIDTelemetry{
		DMIUUID:  "mck-path-dmi",
		DiskID:   "mck-path-disk",
		MAC:      "aa:bb:cc:dd:ee:02",
		CPUModel: "MCK path CPU",
		CPUCores: 4,
	}
	restoreHWID := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry { return tel })
	defer restoreHWID()

	claims := licensing.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	claims.Bind.Mode = "hard"
	claims.HWIDHash = licensing.HashHWIDFromTelemetry(tel)

	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))

	want, err := licensing.DeriveMCK(token, licensing.HostHWID())
	require.NoError(t, err)
	got, err := licensing.DeriveMCKFromLicenseFile(licensePath, pub, licensing.HostFingerprint())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestLoadSealed_InvalidMCK(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	sealed, err := licensing.SealAsset(sealedEdgeAssetLabel, []byte("not-a-real-elf"), mck)
	require.NoError(t, err)

	dir := t.TempDir()
	blob := filepath.Join(dir, "edge_sealed.bin")
	require.NoError(t, os.WriteFile(blob, sealed, 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB", blob)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing-license.jwt"))

	var objs EdgeObjects
	err = loadEdgeObjectsFromSealed(&objs, nil)
	require.Error(t, err)
}
