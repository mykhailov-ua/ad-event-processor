package stream

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type conversionLedgerStore interface {
	UpsertClickConversionLedger(ctx context.Context, arg db.UpsertClickConversionLedgerParams) (db.ClickConversionLedger, error)
}

type ConversionLedgerApplier struct {
	queries conversionLedgerStore
}

func NewConversionLedgerApplier(queries conversionLedgerStore) *ConversionLedgerApplier {
	if queries == nil {
		return nil
	}
	return &ConversionLedgerApplier{queries: queries}
}

func (a *ConversionLedgerApplier) SetStore(queries conversionLedgerStore) {
	if a != nil && queries != nil {
		a.queries = queries
	}
}

func (a *ConversionLedgerApplier) ApplyBatch(ctx context.Context, events []*domain.Event) {
	if a == nil || a.queries == nil || len(events) == 0 {
		return
	}
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
			continue
		}
		clickID := strings.TrimSpace(evt.ClickID)
		if clickID == "" {
			clickID = strings.TrimSpace(track.ParsePayloadStringFields(evt.Payload)["click_id"])
		}
		if clickID == "" || evt.CampaignID == uuid.Nil {
			continue
		}
		delta := extractConversionPayoutDelta(evt.Payload)
		if delta == 0 {
			continue
		}
		status := extractInboundStatus(evt.Payload)
		row, err := a.queries.UpsertClickConversionLedger(ctx, db.UpsertClickConversionLedgerParams{
			CampaignID:  pgtype.UUID{Bytes: evt.CampaignID, Valid: true},
			ClickID:     clickID,
			PayoutMicro: delta,
			LastStatus:  status,
		})
		if err != nil {
			continue
		}
		evt.Payload = mergeConversionLedgerPayload(evt.Payload, row.PayoutMicro, status)
	}
}

func extractConversionPayoutDelta(payload []byte) int64 {
	fields := track.ParsePayloadStringFields(payload)
	for _, key := range []string{"revenue_micro", "payout_micro", "payout"} {
		raw := strings.TrimSpace(fields[key])
		if raw == "" {
			continue
		}
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return v
		}
	}
	return 0
}

func mergeConversionLedgerPayload(original []byte, totalMicro int64, status string) []byte {
	fields := track.ParsePayloadStringFields(original)
	total := strconv.FormatInt(totalMicro, 10)
	fields["conversion_payout_micro"] = total
	fields["revenue_micro"] = total
	fields["payout_micro"] = total
	if status != "" {
		fields["internal_status"] = status
	}
	merged := make(map[string]json.RawMessage, len(fields))
	for k, v := range fields {
		if strings.TrimSpace(v) == "" {
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		merged[k] = b
	}
	if len(merged) == 0 {
		if len(original) > 0 {
			return append([]byte(nil), original...)
		}
		return []byte("{}")
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return original
	}
	return out
}
