package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/pb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMarshalFraudStreamSlot_layerDesyncCount(t *testing.T) {
	slot := &fraudStreamSlot{}
	evt := &domain.Event{
		ClickID:          "click-1",
		CampaignID:       uuid.New(),
		Type:             "click",
		FraudReason:      "tcp_syn_os_mismatch,tls_ja4_mismatch",
		FraudScore:       70,
		LayerDesyncCount: 2,
	}
	fillFraudSlot(slot, 0, evt)

	data, wrap, bufPtr := marshalFraudStreamSlot(slot)
	require.NotNil(t, data)
	require.NotNil(t, wrap)
	require.NotNil(t, bufPtr)
	defer func() {
		byteBufPool.Put(bufPtr)
		byteSliceValuePool.Put(wrap)
	}()

	pbEvt := &pb.AdStreamEvent{}
	require.NoError(t, pbEvt.UnmarshalVT(data))
	assert.Equal(t, uint32(2), pbEvt.LayerDesyncCount)
}
