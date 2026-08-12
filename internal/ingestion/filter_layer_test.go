package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideFraudLayer_L3(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonL3Blocklist)
	assert.Equal(t, FraudLayerL1Reject, decideFraudLayer(acc, FraudTierBlock))
}

func TestDecideFraudLayer_dualL1(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)
	acc.add(FraudReasonLowTTC)
	assert.Equal(t, FraudLayerL1Reject, decideFraudLayer(acc, FraudTierBlock))
}

func TestDecideFraudLayer_singleL1Shadow(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)
	assert.Equal(t, FraudLayerL2Shadow, decideFraudLayer(acc, FraudTierIVT))
}

func TestDecideFraudLayer_weakSignalShadow(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonMissingImpTS)
	assert.Equal(t, FraudLayerL2Shadow, decideFraudLayer(acc, FraudTierSuspect))
}

func TestFraudAccumulator_shortCircuitBudget(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)
	assert.False(t, acc.shouldShortCircuitFraudBudget())

	acc.add(FraudReasonLowTTC)
	assert.True(t, acc.shouldShortCircuitFraudBudget())

	acc.reset()
	acc.add(FraudReasonL3Blocklist)
	assert.True(t, acc.shouldShortCircuitFraudBudget())
}

func TestApplyFraudScoreBoost(t *testing.T) {
	evt := &domain.Event{
		CampaignID: uuid.New(),
	}
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)

	assert.Equal(t, uint32(45), acc.score)

	layer, err := applyFraudLayerDecision(evt, acc, nil, 20)
	assert.NoError(t, err)
	assert.Equal(t, FraudLayerL2Shadow, layer)
	assert.Equal(t, uint32(65), acc.score)
	assert.Equal(t, uint32(65), evt.FraudScore)

	_, err = applyFraudLayerDecision(evt, acc, nil, 20)
	assert.NoError(t, err)
	assert.Equal(t, uint32(65), acc.score)
}

func TestFraudScoreBoost_suspectTierIntegration(t *testing.T) {
	evt := &domain.Event{CampaignID: uuid.New()}
	acc := &fraudAccumulator{
		score:   25,
		count:   1,
		signals: [maxFraudSignals]FraudReasonID{FraudReasonMissingImpTS},
	}

	layer, err := applyFraudLayerDecision(evt, acc, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, FraudLayerL2Shadow, layer)
	assert.Equal(t, FraudTierSuspect, MapFraudTier(uint8(evt.FraudScore), 0, 0, 0, 0))
	assert.Equal(t, uint32(35), evt.FraudScore)
	assert.LessOrEqual(t, evt.FraudScore, uint32(60))
}
