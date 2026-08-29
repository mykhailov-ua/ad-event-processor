package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestMapFraudTier_defaults(t *testing.T) {
	assert.Equal(t, FraudTierPass, MapFraudTier(10, 0, 0, 0, 0))
	assert.Equal(t, FraudTierSuspect, MapFraudTier(45, 30, 60, 80, 100))
	assert.Equal(t, FraudTierIVT, MapFraudTier(70, 30, 60, 80, 100))
	assert.Equal(t, FraudTierBlock, MapFraudTier(90, 30, 60, 80, 100))
}

func TestFraudAccumulator_scoreAndReason(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonDatacenterIP)
	acc.add(FraudReasonLowTTC)
	assert.Equal(t, uint32(90), acc.score)

	evt := &domain.Event{StringBuffer: make([]byte, 0, 64)}
	tier := applyFraudAccumulatorForCampaign(evt, acc, nil)
	assert.Equal(t, FraudTierBlock, tier)
	assert.Equal(t, uint32(90), evt.FraudScore)
	assert.Equal(t, "datacenter_ip,low_ttc", evt.FraudReason)
}

func TestFraudAccumulator_dedupesSignals(t *testing.T) {
	acc := &fraudAccumulator{}
	acc.add(FraudReasonLowTTC)
	acc.add(FraudReasonLowTTC)
	assert.Equal(t, uint8(1), acc.count)
	assert.Equal(t, uint32(45), acc.score)
}

func TestMapFraudTier_campaignThresholds(t *testing.T) {
	camp := &domain.Campaign{
		FraudThresholdPass:    20,
		FraudThresholdSuspect: 40,
		FraudThresholdIVT:     60,
		FraudThresholdBlock:   100,
	}
	pass, suspect, ivt, block := fraudThresholdsFromCampaign(camp)
	assert.Equal(t, FraudTierSuspect, MapFraudTier(25, pass, suspect, ivt, block))
}

func TestAttachFraudAccumulator_reusesWithoutReset(t *testing.T) {
	evt := &domain.Event{}
	acc1 := attachFraudAccumulator(evt)
	acc1.add(FraudReasonIPv4Rotation)
	acc2 := attachFraudAccumulator(evt)
	assert.Equal(t, acc1, acc2)
	assert.True(t, acc2.has(FraudReasonIPv4Rotation))
}

func TestEventHasFraudL3_holdout(t *testing.T) {
	evt := &domain.Event{}
	assert.False(t, eventHasFraudL3(evt))
	acc := attachFraudAccumulator(evt)
	acc.add(FraudReasonDatacenterIP)
	assert.False(t, eventHasFraudL3(evt))
	acc.add(FraudReasonL3Blocklist)
	assert.True(t, eventHasFraudL3(evt))
	releaseFraudAccumulator(evt, acc)
}

func TestEventHasFraudL3_openRTBScratch_holdout(t *testing.T) {
	evt := &domain.Event{}
	slot := acquireOpenRTBScratchSlot()
	attachOpenRTB3Scratch(evt, slot)
	assert.False(t, eventHasFraudL3(evt))
	releaseOpenRTB3Scratch(evt)
}
