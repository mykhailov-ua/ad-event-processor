package postback

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const outboxEventSendPostback = "SEND_POSTBACK"

type conversionPostbackStore interface {
	ListPostbackConfigsByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.PostbackConfig, error)
	ListCampaignsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.Campaign, error)
	CreateOutboxEventsBatch(ctx context.Context, arg db.CreateOutboxEventsBatchParams) error
}

type ConversionPostbackEnqueuer struct {
	queries conversionPostbackStore
}

func NewConversionPostbackEnqueuer(queries conversionPostbackStore) *ConversionPostbackEnqueuer {
	if queries == nil {
		return nil
	}
	return &ConversionPostbackEnqueuer{queries: queries}
}

func (e *ConversionPostbackEnqueuer) SetStore(queries conversionPostbackStore) {
	if e != nil && queries != nil {
		e.queries = queries
	}
}

type pendingConversionEvent struct {
	event      *domain.Event
	campaignID uuid.UUID
}

func (e *ConversionPostbackEnqueuer) OnBatchStored(ctx context.Context, events []*domain.Event) {
	if e == nil || len(events) == 0 {
		return
	}
	pending := make([]pendingConversionEvent, 0, len(events))
	campaignSet := make(map[uuid.UUID]struct{})
	for _, evt := range events {
		if evt == nil || evt.SilentRejectEvent || evt.ShadowEvent || evt.FraudReason != "" {
			continue
		}
		if evt.CampaignID == uuid.Nil || evt.ClickID == "" || evt.Type == "" {
			continue
		}
		pending = append(pending, pendingConversionEvent{event: evt, campaignID: evt.CampaignID})
		campaignSet[evt.CampaignID] = struct{}{}
	}
	if len(pending) == 0 {
		return
	}

	campaignIDs := make([]pgtype.UUID, 0, len(campaignSet))
	for id := range campaignSet {
		campaignIDs = append(campaignIDs, pgtype.UUID{Bytes: id, Valid: true})
	}

	configs, err := e.queries.ListPostbackConfigsByCampaignIDs(ctx, campaignIDs)
	if err != nil {
		slog.Warn("conversion postback batch config load failed", "error", err)
		return
	}
	configByCampaign := make(map[uuid.UUID]db.PostbackConfig, len(configs))
	for i := range configs {
		configByCampaign[uuid.UUID(configs[i].CampaignID.Bytes)] = configs[i]
	}

	campaigns, err := e.queries.ListCampaignsByIDs(ctx, campaignIDs)
	if err != nil {
		slog.Warn("conversion postback batch campaign load failed", "error", err)
		return
	}
	campaignByID := make(map[uuid.UUID]db.Campaign, len(campaigns))
	for i := range campaigns {
		campaignByID[uuid.UUID(campaigns[i].ID.Bytes)] = campaigns[i]
	}

	eventTypes := make([]string, 0, len(pending))
	payloads := make([][]byte, 0, len(pending))
	for i := range pending {
		item := &pending[i]
		cfg, ok := configByCampaign[item.campaignID]
		if !ok {
			continue
		}
		if !eventTypeMatches(item.event.Type, cfg.TargetEvent) {
			continue
		}
		camp, ok := campaignByID[item.campaignID]
		if !ok || !camp.CustomerID.Valid {
			continue
		}
		customerID, err := uuid.FromBytes(camp.CustomerID.Bytes[:])
		if err != nil {
			slog.Warn("conversion postback enqueue failed",
				"campaign_id", item.campaignID,
				"click_id", item.event.ClickID,
				"error", err,
			)
			continue
		}
		payload := buildPostbackPayloadFromEvent(item.event, customerID)
		raw, err := json.Marshal(payload)
		if err != nil {
			slog.Warn("conversion postback enqueue failed",
				"campaign_id", item.campaignID,
				"click_id", item.event.ClickID,
				"error", err,
			)
			continue
		}
		eventTypes = append(eventTypes, outboxEventSendPostback)
		payloads = append(payloads, raw)
	}
	if len(eventTypes) == 0 {
		return
	}
	if err := e.queries.CreateOutboxEventsBatch(ctx, db.CreateOutboxEventsBatchParams{
		EventTypes: eventTypes,
		Payloads:   payloads,
	}); err != nil {
		slog.Warn("conversion postback batch insert failed", "count", len(eventTypes), "error", err)
	}
}

func eventTypeMatches(got, want string) bool {
	got = strings.TrimSpace(strings.ToLower(got))
	want = strings.TrimSpace(strings.ToLower(want))
	if want == "" {
		want = "conversion"
	}
	return got == want
}

func buildPostbackPayloadFromEvent(evt *domain.Event, customerID uuid.UUID) PostbackPayload {
	pb := PostbackPayload{
		CustomerID:  customerID,
		CampaignID:  evt.CampaignID,
		ClickID:     evt.ClickID,
		EventType:   evt.Type,
		PayoutMicro: evt.ClearingPriceMicro,
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

func mergeEventPayloadInto(pb *PostbackPayload, raw []byte) {
	if pb == nil || len(raw) == 0 {
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return
	}
	readString := func(key string) string {
		val, ok := fields[key]
		if !ok || len(val) == 0 {
			return ""
		}
		if val[0] == '"' {
			var s string
			if json.Unmarshal(val, &s) == nil {
				return s
			}
			return ""
		}
		return strings.TrimSpace(string(val))
	}
	if v := readString("fbclid"); v != "" {
		pb.FBCLID = v
	}
	if v := readString("gclid"); v != "" {
		pb.GCLID = v
	}
	if v := readString("ttclid"); v != "" {
		pb.TTCLID = v
	}
	if v := readString("event_source_url"); v != "" {
		pb.EventSourceURL = v
	}
	if v := readString("email"); v != "" {
		pb.Email = v
	}
	if v := readString("phone"); v != "" {
		pb.Phone = v
	}
	if v := readString("tx_id"); v != "" {
		pb.TxID = v
	}
	if v := readString("subid1"); v != "" {
		pb.SubID1 = v
		pb.subSlots[0] = v
	}
	if v := readString("payout_micro"); v != "" {
		var micro int64
		if json.Unmarshal(fields["payout_micro"], &micro) == nil && micro > 0 {
			pb.PayoutMicro = micro
		}
	}
	for i := 1; i <= maxSubMacroSlots; i++ {
		key := subIDJSONKey(i, false)
		if v := readString(key); v != "" {
			pb.subSlots[i-1] = v
			if i == 1 {
				pb.SubID1 = v
			}
			if i == 10 {
				pb.Param10 = v
			}
		}
		key = subIDJSONKey(i, true)
		if v := readString(key); v != "" && pb.subSlots[i-1] == "" {
			pb.subSlots[i-1] = v
			if i == 1 {
				pb.SubID1 = v
			}
			if i == 10 {
				pb.Param10 = v
			}
		}
	}
}
