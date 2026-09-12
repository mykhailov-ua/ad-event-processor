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

type statusSchemeStore interface {
	ListStatusSchemeRulesByCampaignIDs(ctx context.Context, ids []pgtype.UUID) ([]db.CampaignStatusSchemeRule, error)
}

type StatusSchemeApplier struct {
	queries statusSchemeStore
}

func NewStatusSchemeApplier(queries statusSchemeStore) *StatusSchemeApplier {
	if queries == nil {
		return nil
	}
	return &StatusSchemeApplier{queries: queries}
}

func (a *StatusSchemeApplier) SetStore(queries statusSchemeStore) {
	if a != nil && queries != nil {
		a.queries = queries
	}
}

func (a *StatusSchemeApplier) ApplyBatch(ctx context.Context, events []*domain.Event) {
	if a == nil || a.queries == nil || len(events) == 0 {
		return
	}
	campaignSet := make(map[uuid.UUID]struct{})
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
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
	rows, err := a.queries.ListStatusSchemeRulesByCampaignIDs(ctx, ids)
	if err != nil || len(rows) == 0 {
		return
	}
	byCampaign := make(map[uuid.UUID][]db.CampaignStatusSchemeRule, len(campaignSet))
	for i := range rows {
		row := rows[i]
		if !row.CampaignID.Valid {
			continue
		}
		campID := uuid.UUID(row.CampaignID.Bytes)
		byCampaign[campID] = append(byCampaign[campID], row)
	}
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		if domain.ConversionValidationPending(evt.Payload) {
			continue
		}
		rules := byCampaign[evt.CampaignID]
		if len(rules) == 0 {
			continue
		}
		applyStatusSchemeRules(evt, rules)
	}
}

func applyStatusSchemeRules(evt *domain.Event, rules []db.CampaignStatusSchemeRule) {
	if evt == nil || len(rules) == 0 {
		return
	}
	fields := track.ParsePayloadStringFields(evt.Payload)
	inboundStatus := extractInboundStatus(evt.Payload)
	goalName := strings.TrimSpace(fields["goal_name"])
	for i := range rules {
		rule := rules[i]
		if !rule.Enabled {
			continue
		}
		whenStatus := normalizeInboundStatus(rule.WhenStatus)
		if whenStatus != "" && whenStatus != inboundStatus {
			continue
		}
		whenGoal := strings.ToLower(strings.TrimSpace(rule.WhenGoal))
		if whenGoal != "" && whenGoal != strings.ToLower(goalName) {
			continue
		}
		evt.Payload = mergeStatusSchemePayload(evt.Payload, rule)
		return
	}
}

func mergeStatusSchemePayload(original []byte, rule db.CampaignStatusSchemeRule) []byte {
	fields := track.ParsePayloadStringFields(original)
	if v := strings.TrimSpace(rule.SetInternalStatus); v != "" {
		fields["internal_status"] = v
	}
	if v := strings.TrimSpace(rule.SetGoalName); v != "" {
		fields["goal_name"] = v
	}
	switch strings.ToLower(strings.TrimSpace(rule.PayoutMode)) {
	case "fixed":
		fields["revenue_micro"] = strconv.FormatInt(rule.PayoutMicro, 10)
	case "zero":
		fields["revenue_micro"] = "0"
	case "pass_through":
		// keep inbound revenue_micro when present
	case "accumulate_payout":
		if v := strings.TrimSpace(fields["conversion_payout_micro"]); v != "" {
			fields["revenue_micro"] = v
			fields["payout_micro"] = v
		}
	case "inherit", "":
		// keep mapping-applied revenue_micro
	default:
		// unknown mode: no payout change
	}
	if !rule.FireOutbound {
		fields[domain.ConversionStatusSchemeSkipOutboundKey] = "true"
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

func StatusSchemeSkipsOutbound(payload []byte) bool {
	return domain.ConversionSkipsOutboundPostback(payload)
}
