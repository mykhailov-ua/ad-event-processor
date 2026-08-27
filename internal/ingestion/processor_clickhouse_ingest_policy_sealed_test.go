package ingestion

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

func TestProcessorClickHouseIngestPolicy_devModeUsesEmbed(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	raw, err := resolveProcessorClickHouseIngestPolicyBytes()
	require.NoError(t, err)
	require.Equal(t, processorClickHouseIngestPolicyEmbed, raw)
}

func TestProcessorClickHouseIngestPolicy_invalidMCK(t *testing.T) {
	var mck [32]byte
	mck[0] = 1
	sealed, err := licensing.SealAsset(sealedProcessorClickHouseIngestAssetLabel, processorClickHouseIngestPolicyEmbed, mck)
	require.NoError(t, err)

	dir := t.TempDir()
	blob := filepath.Join(dir, "processor_ch_ingest_sealed.bin")
	require.NoError(t, os.WriteFile(blob, sealed, 0o600))
	t.Setenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB", blob)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing-license.jwt"))

	_, err = resolveProcessorClickHouseIngestPolicyBytes()
	require.Error(t, err)
}

func TestProcessorClickHouseIngestPolicy_validMCK(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tel := licensing.HWIDTelemetry{
		DMIUUID:  "proc-ch-policy-dmi",
		DiskID:   "proc-ch-policy-disk",
		MAC:      "aa:bb:cc:dd:ee:03",
		CPUModel: "Processor CH Policy CPU",
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
	wantPolicy := []byte(`{"version":1,"wal_segment_mb_max":256,"compress":false}`)
	sealed, err := licensing.SealAsset(sealedProcessorClickHouseIngestAssetLabel, wantPolicy, mck)
	require.NoError(t, err)

	dir := t.TempDir()
	licensePath := filepath.Join(dir, "license.jwt")
	blobPath := filepath.Join(dir, "processor_ch_ingest_sealed.bin")
	require.NoError(t, os.WriteFile(licensePath, []byte(token), 0o600))
	require.NoError(t, os.WriteFile(blobPath, sealed, 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", licensePath)
	t.Setenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB", blobPath)
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")

	require.NoError(t, InitProcessorClickHouseIngestPolicy())
	policy, ok := ProcessorClickHouseIngestPolicyLoaded()
	require.True(t, ok)
	require.Equal(t, 256, policy.WALSegmentMBMax)
	require.False(t, policy.Compress)

	cfg := DefaultClickHouseSpoolConfig()
	cfg.SegmentSizeBytes = 1024 * 1024 * 1024
	cfg = ApplyClickHouseIngestPolicy(cfg)
	require.Equal(t, int64(256*1024*1024), cfg.SegmentSizeBytes)
}
