package ingest

import (
	"context"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

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
	rot := &filter.RotationSelectContext{
		CampaignID: evt.CampaignID,
		VisitorKey: flowVisitorKey(evt),
		Redis:      h.flowRedisClient(),
		Now:        time.Now(),
	}
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		sel, landerURL, picked := h.campaignFlowTable.SelectForEventExcludingWithRotation(
			evt.CampaignID, UnsafeBytes(uid), evt, exclude, rot,
		)
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
		metrics.OfferClickCapExhaustedTotal.Inc()
		exclude[sel.OfferID] = struct{}{}
	}
	return nil, FlowSelection{}, false
}

func flowVisitorKey(evt *domain.Event) string {
	if evt == nil {
		return ""
	}
	if clickID := strings.TrimSpace(evt.ClickID); clickID != "" {
		return clickID
	}
	if evt.UserID != "" {
		return evt.UserID
	}
	ip := strings.TrimSpace(evt.IP)
	ua := strings.TrimSpace(evt.UA)
	if ip != "" && ua != "" {
		return ip + "|" + ua
	}
	return ""
}

func (h *AdsPacketHandler) flowRedisClient() redis.UniversalClient {
	if h == nil || len(h.redisShards) == 0 {
		return nil
	}
	return h.redisShards[0]
}
