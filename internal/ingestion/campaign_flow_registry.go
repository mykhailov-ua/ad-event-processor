package ingestion

import (
	"sync/atomic"

	"github.com/google/uuid"
)

type campaignFlowRegistrySnapshot struct {
	byCampaign map[uuid.UUID]FlowPathSnapshot
}

// CampaignFlowTable maps campaigns to resolved flow snapshots (GM-M3 RCU).
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

func (t *CampaignFlowTable) Select(campaignID uuid.UUID, userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
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
	return SelectSnapshot(&flow, userID)
}
