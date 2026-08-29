package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLayerDesyncCount_holdoutSingleLayer(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonTCPSynOSMismatch)
	assert.Equal(t, uint8(1), acc.layerDesyncCount())
}

func TestLayerDesyncCount_holdoutMultiLayer(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonTCPSynOSMismatch)
	acc.add(FraudReasonTLSJA4Mismatch)
	acc.add(FraudReasonSecFetchAnomaly)
	assert.Equal(t, uint8(3), acc.layerDesyncCount())
}

func TestLayerDesyncCount_holdoutH2Collapses(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonH2SettingsMismatch)
	acc.add(FraudReasonH2PseudoOrder)
	acc.add(FraudReasonClientHintsMismatch)
	assert.Equal(t, uint8(2), acc.layerDesyncCount())
}

func TestLayerDesyncCount_holdoutNonDesyncSignal(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)
	assert.Equal(t, uint8(0), acc.layerDesyncCount())
}

func TestApplyFraudAccumulatorForCampaign_layerDesyncCount(t *testing.T) {
	evt := &domain.Event{CampaignID: uuid.New()}
	acc := &fraudAccumulator{}
	acc.add(FraudReasonTCPSynOSMismatch)
	acc.add(FraudReasonTLSJA4Mismatch)

	tier := applyFraudAccumulatorForCampaign(evt, acc, nil)
	assert.NotEqual(t, FraudTierPass, tier)
	assert.Equal(t, uint8(2), evt.LayerDesyncCount)
}
