package ingest

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func (h *AdsPacketHandler) selectFlowLanding(evt *domain.Event) (landing []byte, sel FlowSelection, ok bool) {
	return h.selectFlowLandingWithClickCaps(evt)
}

func (h *AdsPacketHandler) selectFlowLandingWithClickCaps(evt *domain.Event) (landing []byte, sel FlowSelection, ok bool) {
	if h == nil || h.campaignFlowTable == nil || evt == nil || evt.CampaignID == uuid.Nil {
		return nil, FlowSelection{}, false
	}
	uid := evt.UserID
	if uid == "" {
		return nil, FlowSelection{}, false
	}
	exclude := make(map[uuid.UUID]struct{})
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		sel, landerURL, picked := h.campaignFlowTable.SelectForEventExcluding(evt.CampaignID, UnsafeBytes(uid), evt, exclude)
		if !picked || len(landerURL) == 0 {
			return nil, FlowSelection{}, false
		}
		if sel.OfferID == uuid.Nil || (sel.CapClicksDaily <= 0 && sel.CapClicksTotal <= 0) {
			return landerURL, sel, true
		}
		reserved, err := filter.ReserveOfferClick(context.Background(), h.flowRedisClient(), sel.OfferID, sel.CapClicksDaily, sel.CapClicksTotal, time.Now())
		if err != nil {
			return nil, FlowSelection{}, false
		}
		if reserved {
			return landerURL, sel, true
		}
		exclude[sel.OfferID] = struct{}{}
	}
	return nil, FlowSelection{}, false
}

func (h *AdsPacketHandler) flowRedisClient() redis.UniversalClient {
	if h == nil || len(h.redisShards) == 0 {
		return nil
	}
	return h.redisShards[0]
}
