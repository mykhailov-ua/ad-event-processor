package ingestion

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type campaignFlowRegistrySnapshot struct {
	byCampaign map[uuid.UUID]FlowPathSnapshot
}

type CampaignFlowTable struct {
	active atomic.Pointer[campaignFlowRegistrySnapshot]
}

func NewCampaignFlowTable() *CampaignFlowTable {
	return &CampaignFlowTable{}
}

func (t *CampaignFlowTable) Publish(snap *campaignFlowRegistrySnapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *CampaignFlowTable) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *CampaignFlowTable) Select(campaignID uuid.UUID, userID []byte, ctx FlowSelectContext) (sel FlowSelection, landerURL []byte, ok bool) {
	if t == nil || campaignID == uuid.Nil || len(userID) == 0 {
		return FlowSelection{}, nil, false
	}
	snap := t.active.Load()
	if snap == nil {
		return FlowSelection{}, nil, false
	}
	flow, ok := snap.byCampaign[campaignID]
	if !ok {
		return FlowSelection{}, nil, false
	}
	return SelectSnapshot(&flow, userID, ctx)
}

func (t *CampaignFlowTable) SelectForEvent(campaignID uuid.UUID, userID []byte, evt *domain.Event) (sel FlowSelection, landerURL []byte, ok bool) {
	return t.Select(campaignID, userID, flowSelectContextFromEvent(evt))
}
