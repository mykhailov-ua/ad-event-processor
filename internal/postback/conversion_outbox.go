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

const outboxEventSendPostback = "SEND_POSTBACK"

type conversionPostbackStore interface {
	ListPostbackConfigsByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.PostbackConfig, error)
	ListOutboundPostbacksByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.CampaignOutboundPostback, error)
	ListCampaignsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.Campaign, error)
	CreatePostbackOutboxEventsBatch(ctx context.Context, arg db.CreatePostbackOutboxEventsBatchParams) error
}

type ConversionPostbackEnqueuer struct {
	queries    conversionPostbackStore
	clickStore ConversionClickStore
}

func NewConversionPostbackEnqueuer(queries any) *ConversionPostbackEnqueuer {
	if queries == nil {
		return nil
	}
	switch q := queries.(type) {
	case *db.Queries:
		return &ConversionPostbackEnqueuer{queries: &conversionPostbackQueries{inner: q}}
	case conversionPostbackStore:
		return &ConversionPostbackEnqueuer{queries: q}
	default:
		panic("postback: unsupported conversion postback store type")
	}
}

func (e *ConversionPostbackEnqueuer) SetClickStore(clicks ConversionClickStore) {
	if e != nil {
		e.clickStore = clicks
	}
}

func (e *ConversionPostbackEnqueuer) SetStore(queries any) {
	if e == nil || queries == nil {
		return
	}
	switch q := queries.(type) {
	case *db.Queries:
		e.queries = &conversionPostbackQueries{inner: q}
	case conversionPostbackStore:
		e.queries = q
	default:
		panic("postback: unsupported conversion postback store type")
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
		if evt == nil || evt.SilentRejectEvent || evt.ShadowEvent || evt.FraudReason != "" || evt.ReviewRoutedEvent {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
			metrics.ConversionPostbackDeferredTotal.Inc()
			continue
		}
		if domain.ConversionSkipsOutboundPostback(evt.Payload) {
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

	outboundRows, err := e.queries.ListOutboundPostbacksByCampaignIDs(ctx, campaignIDs)
	if err != nil {
		slog.Warn("conversion postback batch outbound load failed", "error", err)
		return
	}
	outboundByCampaign := make(map[uuid.UUID][]db.CampaignOutboundPostback, len(campaignSet))
	for i := range outboundRows {
		row := outboundRows[i]
		if !row.CampaignID.Valid {
			continue
		}
		campID := uuid.UUID(row.CampaignID.Bytes)
		outboundByCampaign[campID] = append(outboundByCampaign[campID], row)
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

	reviewRoutedByClick := e.loadReviewRoutedClicks(ctx, pending)

	enqueueAt := time.Now().UTC()
	eventTypes := make([]string, 0, len(pending))
	payloads := make([][]byte, 0, len(pending))
	notBefore := make([]pgtype.Timestamptz, 0, len(pending))
	for i := range pending {
		item := &pending[i]
		if reviewRoutedByClick[item.event.ClickID] {
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
		var legacyCfg *db.PostbackConfig
		if cfg, ok := configByCampaign[item.campaignID]; ok {
			legacyCfg = &cfg
		}
		e.enqueueMatchedPostbacks(ctx, item, customerID, legacyCfg, outboundByCampaign[item.campaignID], enqueueAt, &eventTypes, &payloads, &notBefore)
	}
	if len(eventTypes) == 0 {
		return
	}
	if err := e.queries.CreatePostbackOutboxEventsBatch(ctx, db.CreatePostbackOutboxEventsBatchParams{
		EventTypes: eventTypes,
		Payloads:   payloads,
		NotBefore:  notBefore,
	}); err != nil {
		slog.Warn("conversion postback batch insert failed", "count", len(eventTypes), "error", err)
	}
}

func (e *ConversionPostbackEnqueuer) loadReviewRoutedClicks(ctx context.Context, pending []pendingConversionEvent) map[string]bool {
	if e == nil || e.clickStore == nil || len(pending) == 0 {
		return nil
	}
	clickIDs := make([]string, 0, len(pending))
	seen := make(map[string]struct{}, len(pending))
	for i := range pending {
		id := strings.TrimSpace(pending[i].event.ClickID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clickIDs = append(clickIDs, id)
	}
	if len(clickIDs) == 0 {
		return nil
	}
	snaps, err := e.clickStore.LoadClicks(ctx, clickIDs)
	if err != nil {
		slog.Warn("conversion postback review-routed click lookup failed; fail-open enqueue",
			"error", err, "click_ids", len(clickIDs))
		return nil
	}
	out := make(map[string]bool, len(snaps))
	for id, snap := range snaps {
		if snap.reviewRouted {
			out[id] = true
		}
	}
	return out
}

func eventTypeMatches(got, want string) bool {
	got = strings.TrimSpace(strings.ToLower(got))
	want = strings.TrimSpace(strings.ToLower(want))
	if want == "" {
		want = "conversion"
	}
	return got == want
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
	if v := readString("tblci"); v != "" {
		pb.TBLCI = v
	}
	if v := readString("ob_click_id"); v != "" {
		pb.OBClickID = v
	}
	if v := readString("obclid"); v != "" && pb.OBClickID == "" {
		pb.OBClickID = v
	}
	if v := readString("msclkid"); v != "" {
		pb.MSCLKID = v
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
	if v := readString("event_id"); v != "" {
		pb.EventID = v
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
