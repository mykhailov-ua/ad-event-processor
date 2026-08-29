package stream

import (
	"sort"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type campaignStatRollup struct {
	impressions int64
	clicks      int64
	conversions int64
}

func compactSettlementBatch(events []*domain.Event) ([]*domain.Event, int) {
	if len(events) <= 1 {
		return events, 0
	}
	seen := make(map[string]struct{}, len(events))
	out := make([]*domain.Event, 0, len(events))
	dropped := 0
	for _, evt := range events {
		if evt == nil {
			dropped++
			continue
		}
		key := evt.ClickID + "\x00" + evt.Type
		if _, ok := seen[key]; ok {
			dropped++
			continue
		}
		seen[key] = struct{}{}
		out = append(out, evt)
	}
	return out, dropped
}

func rollupCampaignStats(events []*domain.Event) map[uuid.UUID]campaignStatRollup {
	if len(events) == 0 {
		return nil
	}
	out := make(map[uuid.UUID]campaignStatRollup)
	for _, evt := range events {
		if evt == nil || evt.Type == fraudAggregateEventType {
			continue
		}
		row := out[evt.CampaignID]
		switch evt.Type {
		case "impression":
			row.impressions++
		case "click":
			row.clicks++
		case "conversion":
			if evt.FraudReason != "" {
				continue
			}
			if conversionValidationPending(evt.Payload) {
				continue
			}
			row.conversions++
		default:
			continue
		}
		out[evt.CampaignID] = row
	}
	return out
}

func campaignStatRollupArrays(rollup map[uuid.UUID]campaignStatRollup) (campaignIDs []pgtype.UUID, impressions, clicks, conversions []int64) {
	if len(rollup) == 0 {
		return nil, nil, nil, nil
	}
	ids := make([]uuid.UUID, 0, len(rollup))
	for id := range rollup {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})

	campaignIDs = make([]pgtype.UUID, len(ids))
	impressions = make([]int64, len(ids))
	clicks = make([]int64, len(ids))
	conversions = make([]int64, len(ids))
	for i, id := range ids {
		row := rollup[id]
		campaignIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
		impressions[i] = row.impressions
		clicks[i] = row.clicks
		conversions[i] = row.conversions
	}
	return campaignIDs, impressions, clicks, conversions
}

func adaptiveRefillThresholdPct(emaRPS float64, basePct int) int {
	if basePct <= 0 {
		basePct = 20
	}
	if emaRPS >= 500 {
		if basePct > 10 {
			return basePct / 2
		}
		return 10
	}
	if emaRPS > 0 && emaRPS < 5 {
		capPct := basePct * 2
		if capPct > 40 {
			return 40
		}
		return capPct
	}
	return basePct
}
