package postback

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

type ConversionClickStore interface {
	LoadClicks(ctx context.Context, clickIDs []string) (map[string]clickSnapshot, error)
	LoadExistingGoals(ctx context.Context, keys []conversionGoalKey) (map[conversionGoalKey]struct{}, error)
}

type conversionCampaignReader interface {
	ListSilentRejectByCampaignIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
	ListConversionRejectRulesByCampaignIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.ConversionRejectRules, error)
}

type ConversionDatacenterChecker interface {
	IsDatacenterIP(ip string) bool
}

type ConversionRejectApplier struct {
	cfg       config.ConversionReject
	clicks    ConversionClickStore
	campaigns conversionCampaignReader
	dcCheck   ConversionDatacenterChecker
}

func NewConversionRejectApplier(
	cfg config.ConversionReject,
	clicks ConversionClickStore,
	campaigns conversionCampaignReader,
	dcCheck ConversionDatacenterChecker,
) *ConversionRejectApplier {
	if !cfg.Enabled {
		return nil
	}
	return &ConversionRejectApplier{
		cfg:       cfg,
		clicks:    clicks,
		campaigns: campaigns,
		dcCheck:   dcCheck,
	}
}

func (a *ConversionRejectApplier) ApplyBatch(ctx context.Context, events []*domain.Event) {
	if a == nil || len(events) == 0 {
		return
	}

	conversions := collectConversionCandidates(events)
	if len(conversions) == 0 {
		return
	}

	clickIDs := uniqueClickIDs(conversions)
	clickByID, clickStoreErr := a.loadClicks(ctx, clickIDs)
	degraded := a.clickStoreDegraded(clickStoreErr)
	if clickStoreErr != nil {
		slog.Warn("conversion smart reject click lookup failed; deferring CH-dependent rules",
			"error", clickStoreErr, "click_ids", len(clickIDs))
		metrics.ConversionRejectStoreErrorsTotal.WithLabelValues("load_clicks").Inc()
	}
	if degraded && a.clicks == nil {
		slog.Warn("conversion smart reject click store unavailable; conversions recorded pending validation")
	}

	goalKeys := conversionGoalKeys(conversions)
	existingGoals, goalsStoreErr := a.loadExistingGoals(ctx, goalKeys)
	if goalsStoreErr != nil {
		slog.Warn("conversion smart reject goal lookup failed; in-batch duplicate only",
			"error", goalsStoreErr, "goal_keys", len(goalKeys))
		metrics.ConversionRejectStoreErrorsTotal.WithLabelValues("load_goals").Inc()
		existingGoals = nil
	}

	batchGoals := make(map[conversionGoalKey]struct{}, len(conversions))

	silentByCampaign := a.loadSilentRejectFlags(ctx, conversions)
	rulesByCampaign := a.loadConversionRejectRules(ctx, conversions)

	for _, evt := range conversions {
		if evt == nil || evt.FraudReason != "" || evt.SilentRejectEvent || evt.ShadowEvent {
			continue
		}

		cfg := a.effectiveCfg(evt.CampaignID, rulesByCampaign)
		if !cfg.Enabled {
			continue
		}
		minTTC := time.Duration(cfg.MinTTCDurationMs) * time.Millisecond

		silent := silentByCampaign[evt.CampaignID]

		if degraded {
			if evt.ClickID != "" {
				a.markValidationPending(evt)
				metrics.ConversionRejectDegradedTotal.WithLabelValues("pending_validation").Inc()
			}
			if cfg.RejectDatacenterIP && a.dcCheck != nil && evt.IP != "" {
				if a.dcCheck.IsDatacenterIP(evt.IP) {
					a.reject(evt, ConversionRejectDatacenterIP, silent)
					continue
				}
			}
			continue
		}

		goalName := extractGoalName(evt.Payload)
		goalKey := conversionGoalKey{
			campaignID: evt.CampaignID,
			clickID:    evt.ClickID,
			goalName:   goalName,
		}

		if cfg.RejectNoClick {
			if evt.ClickID == "" {
				a.reject(evt, ConversionRejectNoClick, silent)
				continue
			}
			if _, ok := clickByID[evt.ClickID]; !ok {
				a.reject(evt, ConversionRejectNoClick, silent)
				continue
			}
		}

		click, hasClick := clickByID[evt.ClickID]

		if cfg.RejectLowTTC && hasClick && minTTC > 0 {
			convAt := evt.CreatedAt
			if convAt.IsZero() {
				convAt = time.Now()
			}
			if convAt.Sub(click.createdAt) < minTTC {
				a.reject(evt, ConversionRejectLowTTC, silent)
				continue
			}
		}

		if cfg.RejectDuplicate && goalKey.clickID != "" && goalKey.goalName != "" {
			if _, ok := batchGoals[goalKey]; ok {
				a.reject(evt, ConversionRejectDuplicate, silent)
				continue
			}
			if existingGoals != nil {
				if _, ok := existingGoals[goalKey]; ok {
					a.reject(evt, ConversionRejectDuplicate, silent)
					continue
				}
			}
			batchGoals[goalKey] = struct{}{}
		}

		if cfg.RejectIPDrift && hasClick {
			convCountry := conversionCountry(evt)
			clickCountry := click.country
			if convCountry != "" && clickCountry != "" && convCountry != clickCountry {
				a.reject(evt, ConversionRejectIPDrift, silent)
				continue
			}
		}

		if cfg.RejectDatacenterIP && a.dcCheck != nil && evt.IP != "" {
			if a.dcCheck.IsDatacenterIP(evt.IP) {
				a.reject(evt, ConversionRejectDatacenterIP, silent)
				continue
			}
		}
	}
}

func (a *ConversionRejectApplier) loadClicks(ctx context.Context, clickIDs []string) (map[string]clickSnapshot, error) {
	if a == nil || a.clicks == nil || len(clickIDs) == 0 {
		return nil, nil
	}
	return a.clicks.LoadClicks(ctx, clickIDs)
}

func (a *ConversionRejectApplier) loadExistingGoals(ctx context.Context, keys []conversionGoalKey) (map[conversionGoalKey]struct{}, error) {
	if a == nil || a.clicks == nil || len(keys) == 0 {
		return nil, nil
	}
	return a.clicks.LoadExistingGoals(ctx, keys)
}

func (a *ConversionRejectApplier) clickStoreDegraded(clickStoreErr error) bool {
	if a == nil || a.clicks == nil {
		return true
	}
	return clickStoreErr != nil
}

func (a *ConversionRejectApplier) markValidationPending(evt *domain.Event) {
	if evt == nil {
		return
	}
	evt.Payload = zeroConversionRevenuePayload(evt.Payload)
	evt.Payload = setPayloadBoolFlag(evt.Payload, domain.ConversionValidationPendingKey, true)
}

func (a *ConversionRejectApplier) reject(evt *domain.Event, reason string, silent bool) {
	if evt == nil || reason == "" {
		return
	}
	evt.FraudReason = reason
	if silent {
		evt.SilentRejectEvent = true
	}
	evt.Payload = zeroConversionRevenuePayload(evt.Payload)
}

func (a *ConversionRejectApplier) effectiveCfg(campaignID uuid.UUID, rulesByCampaign map[uuid.UUID]domain.ConversionRejectRules) config.ConversionReject {
	if a == nil {
		return config.ConversionReject{}
	}
	rules, ok := rulesByCampaign[campaignID]
	if !ok {
		return a.cfg
	}
	return mergeConversionRejectConfig(a.cfg, rules)
}

func (a *ConversionRejectApplier) loadConversionRejectRules(ctx context.Context, conversions []*domain.Event) map[uuid.UUID]domain.ConversionRejectRules {
	if a == nil || a.campaigns == nil || len(conversions) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(conversions))
	ids := make([]uuid.UUID, 0, len(conversions))
	for _, evt := range conversions {
		if evt == nil || evt.CampaignID == uuid.Nil {
			continue
		}
		if _, ok := seen[evt.CampaignID]; ok {
			continue
		}
		seen[evt.CampaignID] = struct{}{}
		ids = append(ids, evt.CampaignID)
	}
	if len(ids) == 0 {
		return nil
	}
	rules, err := a.campaigns.ListConversionRejectRulesByCampaignIDs(ctx, ids)
	if err != nil {
		return nil
	}
	return rules
}

func (a *ConversionRejectApplier) loadSilentRejectFlags(ctx context.Context, conversions []*domain.Event) map[uuid.UUID]bool {
	if a == nil || a.campaigns == nil || len(conversions) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(conversions))
	ids := make([]uuid.UUID, 0, len(conversions))
	for _, evt := range conversions {
		if evt == nil || evt.CampaignID == uuid.Nil {
			continue
		}
		if _, ok := seen[evt.CampaignID]; ok {
			continue
		}
		seen[evt.CampaignID] = struct{}{}
		ids = append(ids, evt.CampaignID)
	}
	if len(ids) == 0 {
		return nil
	}
	flags, err := a.campaigns.ListSilentRejectByCampaignIDs(ctx, ids)
	if err != nil {
		return nil
	}
	return flags
}

func collectConversionCandidates(events []*domain.Event) []*domain.Event {
	out := make([]*domain.Event, 0, len(events))
	for _, evt := range events {
		if evt == nil || evt.Type != "conversion" {
			continue
		}
		out = append(out, evt)
	}
	return out
}

func uniqueClickIDs(events []*domain.Event) []string {
	seen := make(map[string]struct{}, len(events))
	out := make([]string, 0, len(events))
	for _, evt := range events {
		if evt == nil || evt.ClickID == "" {
			continue
		}
		if _, ok := seen[evt.ClickID]; ok {
			continue
		}
		seen[evt.ClickID] = struct{}{}
		out = append(out, evt.ClickID)
	}
	return out
}

func conversionGoalKeys(events []*domain.Event) []conversionGoalKey {
	out := make([]conversionGoalKey, 0, len(events))
	for _, evt := range events {
		if evt == nil || evt.ClickID == "" {
			continue
		}
		goal := extractGoalName(evt.Payload)
		if goal == "" {
			continue
		}
		out = append(out, conversionGoalKey{
			campaignID: evt.CampaignID,
			clickID:    evt.ClickID,
			goalName:   goal,
		})
	}
	return out
}

func extractGoalName(payload []byte) string {
	fields := parsePayloadStringFields(payload)
	if v := normalizeGoalName(fields["goal_name"]); v != "" {
		return v
	}
	return normalizeGoalName(fields["goal"])
}

func conversionCountry(evt *domain.Event) string {
	if evt == nil {
		return ""
	}
	if c := normalizeCountryCode(evt.GeoCountry); c != "" {
		return c
	}
	fields := parsePayloadStringFields(evt.Payload)
	if c := normalizeCountryCode(fields["country"]); c != "" {
		return c
	}
	return ""
}

func normalizeCountryCode(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if len(s) != 2 {
		return ""
	}
	return s
}

func normalizeGoalName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func parsePayloadStringFields(payload []byte) map[string]string {
	out := make(map[string]string)
	if len(payload) == 0 {
		return out
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return out
	}
	for key, val := range raw {
		if len(val) == 0 || string(val) == "null" {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err == nil && s != "" {
			out[key] = s
		}
	}
	return out
}

func zeroConversionRevenuePayload(original []byte) []byte {
	fields := parsePayloadStringFields(original)
	if len(fields) == 0 {
		merged := map[string]json.RawMessage{
			"revenue_micro": json.RawMessage("0"),
			"payout_micro":  json.RawMessage("0"),
		}
		out, err := json.Marshal(merged)
		if err != nil {
			return append([]byte(nil), original...)
		}
		return out
	}
	merged := make(map[string]json.RawMessage, len(fields)+2)
	for k, v := range fields {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		merged[k] = b
	}
	merged["revenue_micro"] = json.RawMessage("0")
	merged["payout_micro"] = json.RawMessage("0")
	out, err := json.Marshal(merged)
	if err != nil {
		return append([]byte(nil), original...)
	}
	return out
}

func setPayloadBoolFlag(original []byte, key string, value bool) []byte {
	merged := make(map[string]json.RawMessage)
	if len(original) > 0 {
		if err := json.Unmarshal(original, &merged); err != nil {
			merged = make(map[string]json.RawMessage)
		}
	}
	if value {
		merged[key] = json.RawMessage("true")
	} else {
		merged[key] = json.RawMessage("false")
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return append([]byte(nil), original...)
	}
	return out
}

func clearPayloadBoolFlag(original []byte, key string) []byte {
	merged := make(map[string]json.RawMessage)
	if len(original) > 0 {
		if err := json.Unmarshal(original, &merged); err != nil {
			return append([]byte(nil), original...)
		}
	}
	delete(merged, key)
	if len(merged) == 0 {
		return nil
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return append([]byte(nil), original...)
	}
	return out
}
