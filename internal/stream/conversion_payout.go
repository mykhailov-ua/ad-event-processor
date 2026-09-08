package stream

import (
	"context"
	"encoding/json"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/track"

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

type campaignStatusSchemaStore interface {
	ListAffiliateStatusSchemasByCampaignIDs(ctx context.Context, ids []pgtype.UUID) (map[uuid.UUID]*affiliateStatusSchema, error)
}

type ConversionPayoutApplier struct {
	queries conversionMappingStore
	schemas campaignStatusSchemaStore
}

func NewConversionPayoutApplier(queries conversionMappingStore) *ConversionPayoutApplier {
	if queries == nil {
		return nil
	}
	return &ConversionPayoutApplier{queries: queries}
}

func (a *ConversionPayoutApplier) SetStatusSchemaStore(schemas campaignStatusSchemaStore) {
	if a != nil {
		a.schemas = schemas
	}
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
	if err != nil {
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
	var schemaByCampaign map[uuid.UUID]*affiliateStatusSchema
	if a.schemas != nil {
		schemaByCampaign, _ = a.schemas.ListAffiliateStatusSchemasByCampaignIDs(ctx, ids)
	}
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
			continue
		}
		status := extractInboundStatus(evt.Payload)
		if status == "" {
			continue
		}
		table := byCampaign[evt.CampaignID]
		mapped, ok := table[status]
		if !ok {
			schema := schemaByCampaign[evt.CampaignID]
			if schema == nil {
				continue
			}
			goal, schemaOK := schema.MapAffiliateStatus(status)
			if !schemaOK {
				continue
			}
			evt.Payload = mergeConversionPayoutPayload(evt.Payload, goal, 0)
			continue
		}
		evt.Payload = mergeConversionPayoutPayload(evt.Payload, mapped.goalName, mapped.payoutMicro)
	}
}

func normalizeInboundStatus(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func extractInboundStatus(payload []byte) string {
	fields := track.ParsePayloadStringFields(payload)
	for _, key := range []string{"status", "affiliate_status", "conversion_status", "lead_status"} {
		if v := normalizeInboundStatus(fields[key]); v != "" {
			return v
		}
	}
	return ""
}

func mergeConversionPayoutPayload(original []byte, goalName string, payoutMicro int64) []byte {
	fields := track.ParsePayloadStringFields(original)
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
