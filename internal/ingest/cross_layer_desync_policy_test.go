package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCrossLayerDesyncPolicy_holdoutOffPreservesRoute(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                        campID,
		CrossLayerDesyncAction:    domain.CrossLayerDesyncOff,
		CrossLayerDesyncThreshold: 3,
	})
	evt := &domain.Event{CampaignID: campID, LayerDesyncCount: 4}
	got := evalCrossLayerDesyncClickPolicy(reg, evt)
	assert.False(t, got.Fired)
	assert.False(t, got.ForceSafePage)
	assert.False(t, got.Block)
}

func TestCrossLayerDesyncPolicy_holdoutThreeLayersSafePage(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                        campID,
		CrossLayerDesyncAction:    domain.CrossLayerDesyncSafePage,
		CrossLayerDesyncThreshold: 3,
	})
	evt := &domain.Event{CampaignID: campID, LayerDesyncCount: 3}
	got := evalCrossLayerDesyncClickPolicy(reg, evt)
	assert.True(t, got.Fired)
	assert.True(t, got.ForceSafePage)
	assert.False(t, got.Block)
}

func TestCrossLayerDesyncPolicy_holdoutSingleLayerNoRoute(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                        campID,
		CrossLayerDesyncAction:    domain.CrossLayerDesyncSafePage,
		CrossLayerDesyncThreshold: 3,
	})
	evt := &domain.Event{CampaignID: campID, LayerDesyncCount: 1}
	got := evalCrossLayerDesyncClickPolicy(reg, evt)
	assert.False(t, got.Fired)
	assert.False(t, got.ForceSafePage)
}

func TestCrossLayerDesyncPolicy_holdoutBlockAction(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                        campID,
		CrossLayerDesyncAction:    domain.CrossLayerDesyncBlock,
		CrossLayerDesyncThreshold: 2,
	})
	evt := &domain.Event{CampaignID: campID, LayerDesyncCount: 2}
	got := evalCrossLayerDesyncClickPolicy(reg, evt)
	assert.True(t, got.Fired)
	assert.True(t, got.Block)
}
