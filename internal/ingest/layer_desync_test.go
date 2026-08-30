package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLayerDesyncCount_holdoutSingleLayer(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonTCPSynOSMismatch)
	assert.Equal(t, uint8(1), acc.LayerDesyncCount())
}

func TestLayerDesyncCount_holdoutMultiLayer(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonTCPSynOSMismatch)
	acc.Add(FraudReasonTLSJA4Mismatch)
	acc.Add(FraudReasonSecFetchAnomaly)
	assert.Equal(t, uint8(3), acc.LayerDesyncCount())
}

func TestLayerDesyncCount_holdoutH2Collapses(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonH2SettingsMismatch)
	acc.Add(FraudReasonH2PseudoOrder)
	acc.Add(FraudReasonClientHintsMismatch)
	assert.Equal(t, uint8(2), acc.LayerDesyncCount())
}

func TestLayerDesyncCount_holdoutNonDesyncSignal(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonDatacenterIP)
	assert.Equal(t, uint8(0), acc.LayerDesyncCount())
}

func TestApplyFraudAccumulatorForCampaign_layerDesyncCount(t *testing.T) {
	evt := &domain.Event{CampaignID: uuid.New()}
	acc := &fraudAccumulator{}
	acc.Add(FraudReasonTCPSynOSMismatch)
	acc.Add(FraudReasonTLSJA4Mismatch)

	tier := applyFraudAccumulatorForCampaign(evt, acc, nil)
	assert.NotEqual(t, FraudTierPass, tier)
	assert.Equal(t, uint8(2), evt.LayerDesyncCount)
}
