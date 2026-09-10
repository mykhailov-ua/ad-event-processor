package entitlements

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedGateIngest_couplingOff(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	PublishFeatureSeed(0, false)

	assert.True(t, SeedGateIngest())
}

func TestSeedGateIngest_invalidSeedBlocks(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0, false)

	require.False(t, SeedGateIngest())
}

func TestSeedGateIngest_validSeedAllows(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0x1234_5678, true)

	require.True(t, SeedGateIngest())
}

func TestSeedGateIngest_epochInvalidBlocks_holdout(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(func() {
		ResetFeatureSeedForTest()
		ResetLicenseEpochForTest()
	})
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0x1234_5678, true)
	InvalidateLicenseEpoch()

	require.False(t, SeedGateIngest())
}

func TestSeedGateRPS_zeroMaxStillRequiresSeedCoupling(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0, false)

	require.False(t, SeedGateRPS(0))
}

func TestSeedGateRPS_zeroMaxAllowsWithValidSeed(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(ResetFeatureSeedForTest)
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0x1234_5678, true)

	require.True(t, SeedGateRPS(0))
}

func TestSeedGateRPS_zeroMaxEpochInvalidBlocks_holdout(t *testing.T) {
	ResetFeatureSeedForTest()
	t.Cleanup(func() {
		ResetFeatureSeedForTest()
		ResetLicenseEpochForTest()
	})
	SetSeedCouplingRequired(true)
	PublishFeatureSeed(0x1234_5678, true)
	InvalidateLicenseEpoch()

	require.False(t, SeedGateRPS(0))
}
