package ingestion

import (
	"github.com/google/uuid"
)

func (h *AdsPacketHandler) selectFlowLanding(campaignID uuid.UUID, userID string) (landing []byte, sel FlowSelection, ok bool) {
	if h == nil || h.campaignFlowTable == nil || campaignID == uuid.Nil {
		return nil, FlowSelection{}, false
	}
	uid := userID
	if uid == "" {
		return nil, FlowSelection{}, false
	}
	sel, landerURL, ok := h.campaignFlowTable.Select(campaignID, UnsafeBytes(uid))
	if !ok || len(landerURL) == 0 {
		return nil, FlowSelection{}, false
	}
	return landerURL, sel, true
}
