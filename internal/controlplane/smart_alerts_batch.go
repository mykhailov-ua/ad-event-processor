package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type alertMetricWindowKey struct {
	metric string
	start  time.Time
	end    time.Time
}

type alertCampaignMetrics struct {
	clicks      uint64
	botClicks   uint64
	conversions uint64
	profitMicro int64
	spendMicro  int64
}

func (w *SmartAlertsWorker) evaluateRulesBatch(ctx context.Context, ch *database.CHQuery, rules []smartAlertRuleRow, now time.Time) error {
	if len(rules) == 0 {
		return nil
	}

	campaignsByCustomer, err := w.batchCampaignIDsByCustomer(ctx, rules)
	if err != nil {
		return err
	}

	ruleCampaigns := make([][]uuid.UUID, len(rules))
	metricKeys := make(map[alertMetricWindowKey]map[uuid.UUID]struct{})
	for i, rule := range rules {
		ids, err := resolveRuleCampaignIDs(rule, campaignsByCustomer)
		if err != nil {
			return err
		}
		ruleCampaigns[i] = ids
		if len(ids) == 0 {
			continue
		}
		windowStart, windowEnd := alertWindowBounds(now, rule.WindowMinutes)
		key := alertMetricWindowKey{metric: rule.Metric, start: windowStart, end: windowEnd}
		set := metricKeys[key]
		if set == nil {
			set = make(map[uuid.UUID]struct{})
			metricKeys[key] = set
		}
		for _, id := range ids {
			set[id] = struct{}{}
		}
	}

	metricsByKey, err := w.loadAlertMetricsBatch(ctx, ch, metricKeys)
	if err != nil {
		return err
	}

	existingByWindow := make(map[time.Time]map[uuid.UUID]struct{})
	rulesByWindow := make(map[time.Time][]int)
	for i, rule := range rules {
		windowStart, _ := alertWindowBounds(now, rule.WindowMinutes)
		rulesByWindow[windowStart] = append(rulesByWindow[windowStart], i)
	}
	for windowStart, idxs := range rulesByWindow {
		ruleIDs := make([]pgtype.UUID, len(idxs))
		for j, idx := range idxs {
			ruleIDs[j] = domain.ToUUID(rules[idx].ID)
		}
		rows, err := db.New(w.svc.pool).ListExistingAlertRuleWindows(ctx, db.ListExistingAlertRuleWindowsParams{
			Column1:     ruleIDs,
			WindowStart: pgtype.Timestamptz{Time: windowStart, Valid: true},
		})
		if err != nil {
			return err
		}
		set := make(map[uuid.UUID]struct{}, len(rows))
		for _, row := range rows {
			set[uuid.UUID(row.Bytes)] = struct{}{}
		}
		existingByWindow[windowStart] = set
	}

	for i, rule := range rules {
		campaignIDs := ruleCampaigns[i]
		if len(campaignIDs) == 0 {
			continue
		}
		windowStart, windowEnd := alertWindowBounds(now, rule.WindowMinutes)
		key := alertMetricWindowKey{metric: rule.Metric, start: windowStart, end: windowEnd}
		perCampaign := metricsByKey[key]
		observed := aggregateAlertMetric(rule.Metric, campaignIDs, perCampaign)
		if !alertThresholdBreached(rule.Operator, observed, rule.Threshold) {
			continue
		}
		observed = roundAlertFloat(observed)
		if existing := existingByWindow[windowStart]; existing != nil {
			if _, ok := existing[rule.ID]; ok {
				continue
			}
		}
		if err := w.fireAlertRule(ctx, rule, windowStart, windowEnd, observed); err != nil {
			return err
		}
	}
	return nil
}

func (w *SmartAlertsWorker) batchCampaignIDsByCustomer(ctx context.Context, rules []smartAlertRuleRow) (map[uuid.UUID][]uuid.UUID, error) {
	needCustomer := make(map[uuid.UUID]struct{})
	for _, rule := range rules {
		if !rule.HasCampaign {
			needCustomer[rule.CustomerID] = struct{}{}
		}
	}
	out := make(map[uuid.UUID][]uuid.UUID, len(needCustomer))
	if len(needCustomer) == 0 {
		return out, nil
	}
	customerIDs := make([]pgtype.UUID, 0, len(needCustomer))
	for id := range needCustomer {
		customerIDs = append(customerIDs, domain.ToUUID(id))
	}
	rows, err := db.New(w.svc.pool).ListCampaignIDsByCustomers(ctx, customerIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		customerID := uuid.UUID(row.CustomerID.Bytes)
		campaignID := uuid.UUID(row.CampaignID.Bytes)
		out[customerID] = append(out[customerID], campaignID)
	}
	return out, nil
}

func resolveRuleCampaignIDs(rule smartAlertRuleRow, campaignsByCustomer map[uuid.UUID][]uuid.UUID) ([]uuid.UUID, error) {
	if rule.HasCampaign {
		return []uuid.UUID{rule.CampaignID}, nil
	}
	return campaignsByCustomer[rule.CustomerID], nil
}

func (w *SmartAlertsWorker) loadAlertMetricsBatch(
	ctx context.Context,
	ch *database.CHQuery,
	metricKeys map[alertMetricWindowKey]map[uuid.UUID]struct{},
) (map[alertMetricWindowKey]map[uuid.UUID]alertCampaignMetrics, error) {
	out := make(map[alertMetricWindowKey]map[uuid.UUID]alertCampaignMetrics, len(metricKeys))
	for key, campaignSet := range metricKeys {
		campaignIDs := setToSlice(campaignSet)
		if len(campaignIDs) == 0 {
			continue
		}
		perCampaign, err := querySmartAlertMetricBatch(ctx, ch, key.metric, campaignIDs, key.start, key.end)
		if err != nil {
			return nil, err
		}
		out[key] = perCampaign
	}
	return out, nil
}

func setToSlice(set map[uuid.UUID]struct{}) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func aggregateAlertMetric(metric string, campaignIDs []uuid.UUID, perCampaign map[uuid.UUID]alertCampaignMetrics) float64 {
	var clicks, botClicks, conversions uint64
	var profitMicro, spendMicro int64
	for _, id := range campaignIDs {
		row := perCampaign[id]
		clicks += row.clicks
		botClicks += row.botClicks
		conversions += row.conversions
		profitMicro += row.profitMicro
		spendMicro += row.spendMicro
	}
	switch metric {
	case "clicks":
		return float64(clicks)
	case "bot_clicks":
		return float64(botClicks)
	case "cr":
		if clicks == 0 {
			return 0
		}
		return float64(conversions) / float64(clicks) * 100
	case "roi_pct":
		return CalcROIPct(profitMicro, spendMicro)
	default:
		return 0
	}
}

func (w *SmartAlertsWorker) fireAlertRule(ctx context.Context, rule smartAlertRuleRow, windowStart, windowEnd time.Time, observed float64) error {
	payload := map[string]any{
		"rule_id":        rule.ID.String(),
		"rule_name":      rule.Name,
		"customer_id":    rule.CustomerID.String(),
		"metric":         rule.Metric,
		"operator":       rule.Operator,
		"threshold":      rule.Threshold,
		"observed_value": observed,
		"window_start":   windowStart.Format(time.RFC3339),
		"window_end":     windowEnd.Format(time.RFC3339),
	}
	if rule.HasCampaign {
		payload["campaign_id"] = rule.CampaignID.String()
	}
	payloadBytes, _ := json.Marshal(payload)

	var campParam pgtype.UUID
	if rule.HasCampaign {
		campParam = domain.ToUUID(rule.CampaignID)
	}

	var eventID uuid.UUID
	err := w.svc.pool.QueryRow(ctx, `
		INSERT INTO alert_rule_events (
			rule_id, customer_id, campaign_id, window_start, window_end,
			metric, operator, threshold, observed_value, webhook_status, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', $10)
		RETURNING id`,
		domain.ToUUID(rule.ID), domain.ToUUID(rule.CustomerID), campParam,
		windowStart, windowEnd, rule.Metric, rule.Operator, rule.Threshold, observed,
		payloadBytes,
	).Scan(&eventID)
	if err != nil {
		return fmt.Errorf("insert alert event: %w", err)
	}

	webhookStatus, webhookErr := w.deliverWebhook(ctx, rule.WebhookURL, payloadBytes)
	_, err = w.svc.pool.Exec(ctx, `
		UPDATE alert_rule_events
		SET webhook_status = $2, webhook_error = $3
		WHERE id = $1`,
		domain.ToUUID(eventID), webhookStatus, webhookErr)
	if err != nil {
		return fmt.Errorf("update webhook status: %w", err)
	}
	slog.Info("smart alert fired",
		"rule_id", rule.ID,
		"event_id", eventID,
		"metric", rule.Metric,
		"observed", observed,
		"webhook_status", webhookStatus,
	)
	return nil
}

const (
	smartAlertClicksByCampaignQuery = `
SELECT campaign_id, count() AS n
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id`

	smartAlertBotClicksByCampaignQuery = `
SELECT campaign_id, count() AS n
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id`

	smartAlertConversionsByCampaignQuery = `
SELECT campaign_id, count() AS n
FROM conversions
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id`

	smartAlertROIByCampaignQuery = `
SELECT
 campaign_id,
 sum(revenue_micro) - sum(spend_micro) AS profit_micro,
 sum(spend_micro) AS spend_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id`
)

func querySmartAlertMetricBatch(
	ctx context.Context,
	ch *database.CHQuery,
	metric string,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[uuid.UUID]alertCampaignMetrics, error) {
	chCtx, cancel := context.WithTimeout(ctx, smartAlertCHTimeout)
	defer cancel()

	out := make(map[uuid.UUID]alertCampaignMetrics, len(campaignIDs))
	switch metric {
	case "clicks":
		rows, err := ch.Query(chCtx, smartAlertClicksByCampaignQuery, campaignIDs, from, to)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var campID uuid.UUID
			var n uint64
			if err := rows.Scan(&campID, &n); err != nil {
				return nil, err
			}
			row := out[campID]
			row.clicks = n
			out[campID] = row
		}
		return out, rows.Err()
	case "bot_clicks":
		rows, err := ch.Query(chCtx, smartAlertBotClicksByCampaignQuery, campaignIDs, from, to)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var campID uuid.UUID
			var n uint64
			if err := rows.Scan(&campID, &n); err != nil {
				return nil, err
			}
			row := out[campID]
			row.botClicks = n
			out[campID] = row
		}
		return out, rows.Err()
	case "cr":
		if err := mergeClickCounts(chCtx, ch, smartAlertClicksByCampaignQuery, campaignIDs, from, to, out, false); err != nil {
			return nil, err
		}
		return out, mergeClickCounts(chCtx, ch, smartAlertConversionsByCampaignQuery, campaignIDs, from, to, out, true)
	case "roi_pct":
		rows, err := ch.Query(chCtx, smartAlertROIByCampaignQuery, campaignIDs, from, to)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var campID uuid.UUID
			var profitMicro, spendMicro int64
			if err := rows.Scan(&campID, &profitMicro, &spendMicro); err != nil {
				return nil, err
			}
			row := out[campID]
			row.profitMicro = profitMicro
			row.spendMicro = spendMicro
			out[campID] = row
		}
		return out, rows.Err()
	default:
		return nil, fmt.Errorf("unsupported metric %q", metric)
	}
}

func mergeClickCounts(
	ctx context.Context,
	ch *database.CHQuery,
	query string,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	out map[uuid.UUID]alertCampaignMetrics,
	conversions bool,
) error {
	rows, err := ch.Query(ctx, query, campaignIDs, from, to)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var campID uuid.UUID
		var n uint64
		if err := rows.Scan(&campID, &n); err != nil {
			return err
		}
		row := out[campID]
		if conversions {
			row.conversions = n
		} else {
			row.clicks = n
		}
		out[campID] = row
	}
	return rows.Err()
}
