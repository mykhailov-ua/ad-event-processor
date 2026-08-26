package ingestion

import (
	"context"
	"encoding/json"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type conversionPayoutRow struct {
	goalName    string
	payoutMicro int64
}

type conversionPayoutLookup map[string]conversionPayoutRow

type conversionMappingStore interface {
	ListConversionMappingsByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.CampaignConversionMapping, error)
}

// ConversionPayoutApplier resolves affiliate inbound status to goal_name and revenue_micro on conversion events.
type ConversionPayoutApplier struct {
	queries conversionMappingStore
}

func NewConversionPayoutApplier(queries conversionMappingStore) *ConversionPayoutApplier {
	if queries == nil {
		return nil
	}
	return &ConversionPayoutApplier{queries: queries}
}

func (a *ConversionPayoutApplier) SetStore(queries conversionMappingStore) {
	if a != nil && queries != nil {
		a.queries = queries
	}
}

func (a *ConversionPayoutApplier) ApplyBatch(ctx context.Context, events []*domain.Event) {
	if a == nil || a.queries == nil || len(events) == 0 {
		return
	}
	campaignSet := make(map[uuid.UUID]struct{})
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if evt.CampaignID != uuid.Nil {
			campaignSet[evt.CampaignID] = struct{}{}
		}
	}
	if len(campaignSet) == 0 {
		return
	}
	ids := make([]pgtype.UUID, 0, len(campaignSet))
	for id := range campaignSet {
		ids = append(ids, pgtype.UUID{Bytes: id, Valid: true})
	}
	rows, err := a.queries.ListConversionMappingsByCampaignIDs(ctx, ids)
	if err != nil || len(rows) == 0 {
		return
	}
	byCampaign := make(map[uuid.UUID]conversionPayoutLookup, len(campaignSet))
	for i := range rows {
		row := &rows[i]
		if !row.CampaignID.Valid {
			continue
		}
		campID := uuid.UUID(row.CampaignID.Bytes)
		table := byCampaign[campID]
		if table == nil {
			table = make(conversionPayoutLookup)
			byCampaign[campID] = table
		}
		key := normalizeInboundStatus(row.InboundStatus)
		if key == "" {
			continue
		}
		table[key] = conversionPayoutRow{
			goalName:    row.GoalName,
			payoutMicro: row.PayoutMicro,
		}
	}
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
			continue
		}
		table := byCampaign[evt.CampaignID]
		if len(table) == 0 {
			continue
		}
		status := extractInboundStatus(evt.Payload)
		if status == "" {
			continue
		}
		mapped, ok := table[status]
		if !ok {
			continue
		}
		evt.Payload = mergeConversionPayoutPayload(evt.Payload, mapped.goalName, mapped.payoutMicro)
	}
}

func normalizeInboundStatus(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func extractInboundStatus(payload []byte) string {
	fields := parsePayloadStringFields(payload)
	for _, key := range []string{"status", "affiliate_status", "conversion_status"} {
		if v := normalizeInboundStatus(fields[key]); v != "" {
			return v
		}
	}
	return ""
}

func mergeConversionPayoutPayload(original []byte, goalName string, payoutMicro int64) []byte {
	fields := parsePayloadStringFields(original)
	if goalName != "" {
		fields["goal_name"] = goalName
	}
	merged := make(map[string]json.RawMessage, len(fields)+1)
	for k, v := range fields {
		if v == "" {
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		merged[k] = b
	}
	if payoutMicro > 0 {
		b, err := json.Marshal(payoutMicro)
		if err == nil {
			merged["revenue_micro"] = b
		}
	}
	if len(merged) == 0 {
		if len(original) > 0 {
			return append([]byte(nil), original...)
		}
		return nil
	}
	out, err := json.Marshal(merged)
	if err != nil {
		if len(original) > 0 {
			return append([]byte(nil), original...)
		}
		return nil
	}
	return out
}
