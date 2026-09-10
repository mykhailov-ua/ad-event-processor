package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVerifySealedAssetsReady_devModeSkips(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	require.NoError(t, VerifySealedAssetsReady(SealedAssetRoleTracker))
}

func TestVerifySealedAssetsReady_probeDisabledSkips(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "development")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "0")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(t.TempDir(), "missing.jwt"))
	require.NoError(t, VerifySealedAssetsReady(SealedAssetRoleTracker))
}

func TestVerifySealedAssetsReady_licenseRequiredMissingFile(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_PROFILE", "production")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_REQUIRED", "1")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(t.TempDir(), "missing.jwt"))

	err := VerifySealedAssetsReady(SealedAssetRoleTracker)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestVerifySealedAssetsReady_trackerMissingBlob(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing-unified_filter_sealed.bin")
	t.Setenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB", missing)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "license.jwt"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "license.jwt"), []byte("token"), 0o600))

	err := VerifySealedAssetsReady(SealedAssetRoleTracker)
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestVerifySealedAssetsReady_trackerInvalidMCK(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	sealed, err := SealAsset(AssetLabelUnifiedFilter, []byte("-- lua"), mck)
	require.NoError(t, err)

	dir := t.TempDir()
	blob := filepath.Join(dir, "unified_filter_sealed.bin")
	require.NoError(t, os.WriteFile(blob, sealed, 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB", blob)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing-license.jwt"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "missing-license.jwt"), []byte("not-a-jwt"), 0o600))

	err = VerifySealedAssetsReady(SealedAssetRoleTracker)
	require.Error(t, err)
}

func TestVerifySealedAssetsReady_trackerValidMCK(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := HWIDTelemetry{
		DMIUUID:  "sealed-startup-dmi",
		DiskID:   "sealed-startup-disk",
		MAC:      "aa:bb:cc:dd:ee:05",
		CPUModel: "Sealed Startup CPU",
		CPUCores: 4,
	}
	restore := SetHWIDCollectForTest(func() HWIDTelemetry { return tel })
	defer restore()

	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	claims.Bind.Mode = "hard"
	claims.HWIDHash = HashHWIDFromTelemetry(tel)

	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	require.NoError(t, err)

	mck, err := DeriveMCK(token, HostHWID())
	require.NoError(t, err)
	wantLua := "-- sealed startup test\nreturn 0\n"
	sealed, err := SealAsset(AssetLabelUnifiedFilter, []byte(wantLua), mck)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	blobPath := filepath.Join(dir, "unified_filter_sealed.bin")
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))
	require.NoError(t, os.WriteFile(blobPath, sealed, 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", licensePath)
	t.Setenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB", blobPath)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	require.NoError(t, VerifySealedAssetsReady(SealedAssetRoleTracker))
}

func TestVerifySealedAssetsReady_processorOptionalBlobMissing(t *testing.T) {
	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(licensePath, []byte("token"), 0o600))
	missing := filepath.Join(dir, "missing-processor_ch_ingest_sealed.bin")

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", licensePath)
	t.Setenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB", missing)

	require.NoError(t, VerifySealedAssetsReady(SealedAssetRoleProcessor))
}

func TestVerifySealedAssetsReady_edgeOptionalBlobMissing(t *testing.T) {
	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	require.NoError(t, os.WriteFile(licensePath, []byte("token"), 0o600))
	missing := filepath.Join(dir, "missing-edge_sealed.bin")

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", licensePath)
	t.Setenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB", missing)

	require.NoError(t, VerifySealedAssetsReady(SealedAssetRoleEdge))
}
