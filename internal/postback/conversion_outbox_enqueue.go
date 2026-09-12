package postback

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (e *ConversionPostbackEnqueuer) enqueueMatchedPostbacks(
	ctx context.Context,
	item *pendingConversionEvent,
	customerID uuid.UUID,
	legacyCfg *db.PostbackConfig,
	outbounds []db.CampaignOutboundPostback,
	enqueueAt time.Time,
	eventTypes *[]string,
	payloads *[][]byte,
	notBefore *[]pgtype.Timestamptz,
) {
	if item == nil || item.event == nil {
		return
	}
	if len(outbounds) > 0 {
		for i := range outbounds {
			row := outbounds[i]
			if !OutboundPostbackMatchesEvent(row, item.event) {
				continue
			}
			postbackID := uuid.UUID(row.ID.Bytes)
			if !ShouldFireOutboundPostbackSample(item.event.ClickID, postbackID, row.SamplePercent) {
				metrics.PostbackSampledSkippedTotal.Inc()
				continue
			}
			e.appendPostbackOutbox(
				ctx,
				item,
				customerID,
				buildPostbackPayloadFromEvent(item.event, customerID, postbackID),
				row.Provider,
				row.DelaySeconds,
				enqueueAt,
				eventTypes,
				payloads,
				notBefore,
			)
		}
		return
	}
	if legacyCfg == nil {
		return
	}
	if !eventTypeMatches(item.event.Type, legacyCfg.TargetEvent) {
		return
	}
	payload := buildPostbackPayloadFromEvent(item.event, customerID, uuid.Nil)
	e.appendPostbackOutbox(ctx, item, customerID, payload, legacyCfg.Provider, 0, enqueueAt, eventTypes, payloads, notBefore)
}

func (e *ConversionPostbackEnqueuer) appendPostbackOutbox(
	ctx context.Context,
	item *pendingConversionEvent,
	customerID uuid.UUID,
	payload PostbackPayload,
	provider string,
	delaySeconds int32,
	enqueueAt time.Time,
	eventTypes *[]string,
	payloads *[][]byte,
	notBefore *[]pgtype.Timestamptz,
) {
	if capiPostbackProvider(provider) && strings.TrimSpace(payload.EventID) == "" {
		metrics.ConversionBrowserMissingTotal.Inc()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("conversion postback enqueue failed",
			"campaign_id", item.campaignID,
			"click_id", item.event.ClickID,
			"error", err,
		)
		return
	}
	*eventTypes = append(*eventTypes, outboxEventSendPostback)
	*payloads = append(*payloads, raw)
	*notBefore = append(*notBefore, postbackNotBeforePG(delaySeconds, enqueueAt))
}

func postbackNotBeforePG(delaySeconds int32, base time.Time) pgtype.Timestamptz {
	if ClampOutboundDelaySeconds(delaySeconds) <= 0 {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: PostbackNotBefore(delaySeconds, base), Valid: true}
}

func buildPostbackPayloadFromEvent(evt *domain.Event, customerID uuid.UUID, outboundPostbackID uuid.UUID) PostbackPayload {
	pb := PostbackPayload{
		CustomerID:         customerID,
		CampaignID:         evt.CampaignID,
		ClickID:            evt.ClickID,
		EventType:          evt.Type,
		PayoutMicro:        evt.ClearingPriceMicro,
		OutboundPostbackID: outboundPostbackID,
	}
	mergeEventPayloadInto(&pb, evt.Payload)
	if pb.TxID == "" {
		pb.TxID = evt.ClickID
	}
	if pb.EventSourceURL == "" {
		pb.EventSourceURL = synthesizeEventSourceURL(pb, "")
	}
	return pb
}
