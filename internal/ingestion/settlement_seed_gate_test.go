package ingestion

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
)

func TestSettlementSeedGateAllowed_blocksInvalidSeed(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0, false)

	assert.False(t, licensing.SettlementSeedGateAllowed())
}

func TestSettlementSeedGateAllowed_allowsValidSeed(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.SetSeedCouplingRequired(true)
	licensing.PublishFeatureSeed(0x1234_5678, true)

	assert.True(t, licensing.SettlementSeedGateAllowed())
}

func TestSettlementSeedGateAllowed_skipsWhenCouplingOff(t *testing.T) {
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.PublishFeatureSeed(0, false)

	assert.True(t, licensing.SettlementSeedGateAllowed())
}
