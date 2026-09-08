package netintel

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResidentialMobileTier_holdoutBuiltinMVNO(t *testing.T) {
	table := NewMobileTierTable()
	assert.Equal(t, uint8(4), table.MobileTier(310410))
	assert.Equal(t, uint8(3), table.MobileTier(26615))
	assert.Equal(t, uint8(0), table.MobileTier(64512))
}

func TestResidentialMobileTier_holdoutFeedMergeOverridesBuiltin(t *testing.T) {
	table := NewMobileTierTable()
	table.Publish(map[uint32]uint8{310410: 1}, 2)
	assert.Equal(t, uint8(1), table.MobileTier(310410))
}

func TestResidentialMobileTier_holdoutParseFeedLine(t *testing.T) {
	asn, tier, ok := parseMobileTierLine("AS58453 4")
	require.True(t, ok)
	assert.Equal(t, uint32(58453), asn)
	assert.Equal(t, uint8(4), tier)
}

func TestResidentialMobileProbeRisk_holdoutWeightedSum(t *testing.T) {
	weights := MobileProbeRiskWeights{Alpha: 1, Beta: 0.5, Gamma: 0.25}
	risk := ComputeMobileProbeRisk(weights, 4, 80, 1)
	assert.InDelta(t, 4+40+25, risk, 0.001)
}

func TestResidentialProxyRing_peekWithoutObserveIncrement(t *testing.T) {
	ring := NewResidentialProxyRing()
	cid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	campaignHash := domain.CRC32Castagnoli(&cid)
	ring.SeedForTest(cid, ResidentialProxyRow{
		Events:      150,
		Clicks:      5,
		UniqueUsers: 30,
		UniqueUAs:   12,
	})
	row, signal := ring.Peek(campaignHash)
	assert.Equal(t, 150, row.Events)
	assert.True(t, signal)
	row2, _ := ring.Peek(campaignHash)
	assert.Equal(t, row.Events, row2.Events)
}
