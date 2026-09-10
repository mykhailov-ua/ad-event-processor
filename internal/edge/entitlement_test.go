package edge

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEbpfEdgeLicensed_missingKeyFailsOpen(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	assert.True(t, EbpfEdgeLicensed(context.Background(), redisClient))
}

func TestEbpfEdgeLicensed_enabled(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	mr.HSet(entitlementDeploymentKey, entitlementEbpfXDPEdge, "1")

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	assert.True(t, EbpfEdgeLicensed(context.Background(), redisClient))
}

func TestEbpfEdgeLicensed_deniedWhenZero(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	mr.HSet(entitlementDeploymentKey, entitlementEbpfXDPEdge, "0")

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	assert.False(t, EbpfEdgeLicensed(context.Background(), redisClient))
}

func TestEbpfEdgeLicensed_nilClientFailsOpen(t *testing.T) {
	assert.True(t, EbpfEdgeLicensed(context.Background(), nil))
}

func TestEbpfEdgeSeedGateAllowed_devUnsealed(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	assert.True(t, EbpfEdgeSeedGateAllowed())
}

func TestEbpfEdgeSeedGateAllowed_enterpriseMissingLicense(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)

	dir := t.TempDir()
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", filepath.Join(dir, "missing.jwt"))

	assert.False(t, EbpfEdgeSeedGateAllowed())
}

func TestEbpfEdgeSeedGateAllowed_seedCouplingDisabled(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_SEED_COUPLE", "0")
	assert.True(t, EbpfEdgeSeedGateAllowed())
}

func TestEbpfEdgeSeedGateAllowed_starterSKUBlocksEbpf(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(entitlements.ResetFeatureSeedForTest)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		SKU:          entitlements.SKUCodeStarter,
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		Features: entitlements.FeatureSet{
			EbpfXDPEdge: true,
		},
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", path)

	assert.False(t, EbpfEdgeSeedGateAllowed())
}

func TestEbpfEdgeSeedGateAllowed_epochInvalidBlocks(t *testing.T) {
	entitlements.ResetFeatureSeedForTest()
	t.Cleanup(func() {
		entitlements.ResetFeatureSeedForTest()
		entitlements.ResetLicenseEpochForTest()
	})

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "license.jwt")
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      uuid.NewString(),
		DeploymentID: uuid.NewString(),
		SKU:          entitlements.SKUCodeEnterprise,
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		Features: entitlements.FeatureSet{
			EbpfXDPEdge: true,
		},
	}
	token, err := licensing.SignJWT(claims, priv, licensing.DefaultLicenseKeyID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "enterprise")
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY", hex.EncodeToString(pub))
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_PATH", path)

	entitlements.InvalidateLicenseEpoch()
	assert.False(t, EbpfEdgeSeedGateAllowed())
}
