package postback

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const outboxEventSendPostback = "SEND_POSTBACK"

type conversionPostbackStore interface {
	GetPostbackConfig(ctx context.Context, id pgtype.UUID) (db.PostbackConfig, error)
	GetCampaign(ctx context.Context, id pgtype.UUID) (db.Campaign, error)
	CreateOutboxEvent(ctx context.Context, arg db.CreateOutboxEventParams) (db.OutboxEvent, error)
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

func (e *ConversionPostbackEnqueuer) OnBatchStored(ctx context.Context, events []*domain.Event) {
	if e == nil || len(events) == 0 {
		return
	}
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if err := e.enqueueOne(ctx, evt); err != nil {
			slog.Warn("conversion postback enqueue failed",
				"campaign_id", evt.CampaignID,
				"click_id", evt.ClickID,
				"event_type", evt.Type,
				"error", err,
			)
		}
	}
}

func (e *ConversionPostbackEnqueuer) enqueueOne(ctx context.Context, evt *domain.Event) error {
	if evt.GhostEvent || evt.ShadowEvent || evt.FraudReason != "" {
		return nil
	}
	if evt.CampaignID == uuid.Nil || evt.ClickID == "" || evt.Type == "" {
		return nil
	}

	cfg, err := e.queries.GetPostbackConfig(ctx, pgtype.UUID{Bytes: evt.CampaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if !eventTypeMatches(evt.Type, cfg.TargetEvent) {
		return nil
	}

	camp, err := e.queries.GetCampaign(ctx, pgtype.UUID{Bytes: evt.CampaignID, Valid: true})
	if err != nil {
		return err
	}
	if !camp.CustomerID.Valid {
		return nil
	}
	customerID, err := uuid.FromBytes(camp.CustomerID.Bytes[:])
	if err != nil {
		return err
	}

	payload := buildPostbackPayloadFromEvent(evt, customerID)
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = e.queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: outboxEventSendPostback,
		Payload:   raw,
	})
	return err
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
