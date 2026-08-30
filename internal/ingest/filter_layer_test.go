package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideFraudLayer_L3(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonL3Blocklist)
	assert.Equal(t, FraudLayerL1Reject, decideFraudLayer(acc, FraudTierBlock))
}

func TestDecideFraudLayer_dualL1(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonDatacenterIP)
	acc.Add(FraudReasonLowTTC)
	assert.Equal(t, FraudLayerL1Reject, decideFraudLayer(acc, FraudTierBlock))
}

func TestDecideFraudLayer_singleL1Shadow(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonDatacenterIP)
	assert.Equal(t, FraudLayerL2Shadow, decideFraudLayer(acc, FraudTierIVT))
}

func TestDecideFraudLayer_weakSignalShadow(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonMissingImpTS)
	assert.Equal(t, FraudLayerL2Shadow, decideFraudLayer(acc, FraudTierSuspect))
}

func TestFraudAccumulator_shortCircuitBudget(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonDatacenterIP)
	assert.False(t, acc.ShouldShortCircuitFraudBudget())

	acc.Add(FraudReasonLowTTC)
	assert.True(t, acc.ShouldShortCircuitFraudBudget())

	acc.Reset()
	acc.Add(FraudReasonL3Blocklist)
	assert.True(t, acc.ShouldShortCircuitFraudBudget())
}

func TestApplyFraudScoreBoost(t *testing.T) {
	evt := &domain.Event{
		CampaignID: uuid.New(),
	}
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonDatacenterIP)

	assert.Equal(t, uint32(45), acc.Score())

	layer, err := applyFraudLayerDecision(evt, acc, nil, 20)
	assert.NoError(t, err)
	assert.Equal(t, FraudLayerL2Shadow, layer)
	assert.Equal(t, uint32(65), acc.Score())
	assert.Equal(t, uint32(65), evt.FraudScore)

	_, err = applyFraudLayerDecision(evt, acc, nil, 20)
	assert.NoError(t, err)
	assert.Equal(t, uint32(65), acc.Score())
}

func TestFraudScoreBoost_suspectTierIntegration(t *testing.T) {
	evt := &domain.Event{CampaignID: uuid.New()}
	acc := newFraudAccumulatorForTest(25, FraudReasonMissingImpTS)

	layer, err := applyFraudLayerDecision(evt, acc, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, FraudLayerL2Shadow, layer)
	assert.Equal(t, FraudTierSuspect, MapFraudTier(uint8(evt.FraudScore), 0, 0, 0, 0))
	assert.Equal(t, uint32(35), evt.FraudScore)
	assert.LessOrEqual(t, evt.FraudScore, uint32(60))
}
