package ingest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveUnifiedFilterLua_devModeUsesEmbed(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	src, err := resolveUnifiedFilterLuaSource()
	require.NoError(t, err)
	require.Equal(t, unifiedFilterLua, src)
}

func TestResolveUnifiedFilterLua_invalidMCK(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	sealed, err := licensing.SealAsset(sealedUnifiedFilterAssetLabel, []byte("-- lua"), mck)
	require.NoError(t, err)

	dir := t.TempDir()
	blob := filepath.Join(dir, "unified_filter_sealed.bin")
	require.NoError(t, os.WriteFile(blob, sealed, 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB", blob)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing-license.jwt"))

	_, err = resolveUnifiedFilterLuaSource()
	require.Error(t, err)
}

func TestResolveUnifiedFilterLua_validMCK(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := licensing.HWIDTelemetry{
		DMIUUID:  "lua-seal-dmi",
		DiskID:   "lua-seal-disk",
		MAC:      "aa:bb:cc:dd:ee:02",
		CPUModel: "Lua Seal CPU",
		CPUCores: 4,
	}
	restore := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry { return tel })
	defer restore()

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

	mck, err := licensing.DeriveMCK(token, licensing.HostHWID())
	require.NoError(t, err)
	wantLua := "-- sealed unified-filter test\nreturn 0\n"
	sealed, err := licensing.SealAsset(sealedUnifiedFilterAssetLabel, []byte(wantLua), mck)
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

	got, err := resolveUnifiedFilterLuaSource()
	require.NoError(t, err)
	require.Equal(t, wantLua, got)
}
