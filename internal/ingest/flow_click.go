package ingest

import (
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func (h *AdsPacketHandler) selectFlowLanding(evt *domain.Event) (landing []byte, sel FlowSelection, ok bool) {
	if h == nil || h.campaignFlowTable == nil || evt == nil || evt.CampaignID == uuid.Nil {
		return nil, FlowSelection{}, false
	}
	uid := evt.UserID
	if uid == "" {
		return nil, FlowSelection{}, false
	}
	sel, landerURL, ok := h.campaignFlowTable.SelectForEvent(evt.CampaignID, UnsafeBytes(uid), evt)
	if !ok || len(landerURL) == 0 {
		return nil, FlowSelection{}, false
	}
	return landerURL, sel, true
}
