package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramReportSummary struct {
	Clicks      int64 `json:"clicks"`
	Impressions int64 `json:"impressions"`
	Premium     int64 `json:"premium"`
	Motivated   int64 `json:"motivated"`
	Conversions int64 `json:"conversions"`
	Funnel      struct {
		Validates int64 `json:"validates"`
		Clicks    int64 `json:"clicks"`
		Motivated int64 `json:"motivated"`
		Premium   int64 `json:"premium"`
	} `json:"funnel"`
	Freshness ReportFreshness `json:"freshness"`
}

type ReportFreshness struct {
	Stale      bool  `json:"stale"`
	LagSeconds int64 `json:"lag_seconds"`
}

type TelegramFunnelRow struct {
	StartParam  string `json:"start_param"`
	Clicks      int64  `json:"clicks"`
	Impressions int64  `json:"impressions"`
	Conversions int64  `json:"conversions"`
}

type TelegramFunnelReport struct {
	Rows      []TelegramFunnelRow `json:"rows"`
	Freshness ReportFreshness     `json:"freshness"`
}

type TelegramBotBreakdownRow struct {
	BotID       uint64 `json:"bot_id"`
	Clicks      int64  `json:"clicks"`
	Impressions int64  `json:"impressions"`
	Premium     int64  `json:"premium"`
}

type TelegramBotsReport struct {
	Rows      []TelegramBotBreakdownRow `json:"rows"`
	Freshness ReportFreshness           `json:"freshness"`
}

type TelegramPremiumReport struct {
	PremiumClicks    int64           `json:"premium_clicks"`
	NonPremiumClicks int64           `json:"non_premium_clicks"`
	PremiumRatePct   float64         `json:"premium_rate_pct"`
	Freshness        ReportFreshness `json:"freshness"`
}

type TelegramFraudReport struct {
	BlockedClicks int64           `json:"blocked_clicks"`
	ShadowClicks  int64           `json:"shadow_clicks"`
	Freshness     ReportFreshness `json:"freshness"`
}

func (s *Service) reportFreshness(ctx context.Context) ReportFreshness {
	fresh := ReportFreshness{}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		fresh.Stale = true
		return fresh
	}
	lag, err := clickhouseQuery.IngestionLag(ctx)
	if err != nil {
		fresh.Stale = true
		return fresh
	}
	fresh.LagSeconds = int64(lag.Seconds())
	fresh.Stale = lag > 5*time.Minute
	return fresh
}

func listCustomerCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	rows, err := pool.Query(ctx, `SELECT id FROM campaigns WHERE customer_id = $1 AND deleted_at IS NULL`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Service) resolveTelegramCampaignIDs(
	ctx context.Context,
	filter ReportFilter,
) ([]uuid.UUID, error) {
	if filter.CampaignID != nil {
		return []uuid.UUID{*filter.CampaignID}, nil
	}
	if filter.CustomerID != nil {
		return listCustomerCampaignIDs(ctx, s.pool, *filter.CustomerID)
	}
	return nil, nil
}

func appendTelegramCampaignFilter(query string, args []any, campaignIDs []uuid.UUID) (string, []any, bool) {
	if campaignIDs == nil {
		return query, args, true
	}
	if len(campaignIDs) == 0 {
		return query, args, false
	}
	return query + ` AND campaign_id IN (?)`, append(args, campaignIDs), true
}

func (s *Service) queryTelegramCounts(
	ctx context.Context,
	from, to time.Time,
	campaignIDs []uuid.UUID,
) (clicks, impressions, premium, motivated, conversions int64, err error) {
	if campaignIDs != nil && len(campaignIDs) == 0 {
		return 0, 0, 0, 0, 0, nil
	}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return 0, 0, 0, 0, 0, errors.New("clickhouse connection not available")
	}
	query := `
		SELECT
			countIf(event_type = 'tg_click') AS clicks,
			countIf(event_type = 'tg_impression') AS impressions,
			countIf(is_premium = 1) AS premium,
			countIf(motivated = 1) AS motivated,
			countIf(event_type = 'tg_conversion') AS conversions
		FROM tg_events
		WHERE created_at >= ? AND created_at < ?`
	args := []any{from, to}
	var ok bool
	query, args, ok = appendTelegramCampaignFilter(query, args, campaignIDs)
	if !ok {
		return 0, 0, 0, 0, 0, nil
	}
	err = clickhouseQuery.QueryRow(ctx, query, args...).Scan(&clicks, &impressions, &premium, &motivated, &conversions)
	return clicks, impressions, premium, motivated, conversions, err
}

func (s *Service) GetTelegramSummaryReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	campaignIDs, err := s.resolveTelegramCampaignIDs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("resolve campaign filter: %w", err)
	}
	clicks, impressions, premium, motivated, conversions, err := s.queryTelegramCounts(ctx, from, to, campaignIDs)
	if err != nil {
		return nil, fmt.Errorf("clickhouse query failed: %w", err)
	}
	report := TelegramReportSummary{
		Clicks:      clicks,
		Impressions: impressions,
		Premium:     premium,
		Motivated:   motivated,
		Conversions: conversions,
		Freshness:   s.reportFreshness(ctx),
	}
	report.Funnel.Validates = clicks
	report.Funnel.Clicks = clicks
	report.Funnel.Motivated = motivated
	report.Funnel.Premium = premium
	return json.Marshal(report)
}

func (s *Service) GetTelegramFunnelReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	campaignIDs, err := s.resolveTelegramCampaignIDs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("resolve campaign filter: %w", err)
	}
	if campaignIDs != nil && len(campaignIDs) == 0 {
		return json.Marshal(TelegramFunnelReport{Freshness: s.reportFreshness(ctx)})
	}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return nil, errors.New("clickhouse connection not available")
	}
	query := `
		SELECT
			start_param,
			countIf(event_type = 'tg_click') AS clicks,
			countIf(event_type = 'tg_impression') AS impressions,
			countIf(event_type = 'tg_conversion') AS conversions
		FROM tg_events
		WHERE created_at >= ? AND created_at < ?`
	args := []any{from, to}
	var ok bool
	query, args, ok = appendTelegramCampaignFilter(query, args, campaignIDs)
	if !ok {
		return json.Marshal(TelegramFunnelReport{Freshness: s.reportFreshness(ctx)})
	}
	query += ` GROUP BY start_param ORDER BY clicks DESC LIMIT 100`

	rows, err := clickhouseQuery.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse funnel query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	report := TelegramFunnelReport{Freshness: s.reportFreshness(ctx)}
	for rows.Next() {
		var row TelegramFunnelRow
		if err := rows.Scan(&row.StartParam, &row.Clicks, &row.Impressions, &row.Conversions); err != nil {
			return nil, err
		}
		report.Rows = append(report.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(report)
}

func (s *Service) GetTelegramBotsReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	campaignIDs, err := s.resolveTelegramCampaignIDs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("resolve campaign filter: %w", err)
	}
	if campaignIDs != nil && len(campaignIDs) == 0 {
		return json.Marshal(TelegramBotsReport{Freshness: s.reportFreshness(ctx)})
	}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return nil, errors.New("clickhouse connection not available")
	}
	query := `
		SELECT
			bot_id,
			countIf(event_type = 'tg_click') AS clicks,
			countIf(event_type = 'tg_impression') AS impressions,
			countIf(is_premium = 1) AS premium
		FROM tg_events
		WHERE created_at >= ? AND created_at < ? AND bot_id > 0`
	args := []any{from, to}
	var ok bool
	query, args, ok = appendTelegramCampaignFilter(query, args, campaignIDs)
	if !ok {
		return json.Marshal(TelegramBotsReport{Freshness: s.reportFreshness(ctx)})
	}
	query += ` GROUP BY bot_id ORDER BY clicks DESC LIMIT 50`

	rows, err := clickhouseQuery.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse bots query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	report := TelegramBotsReport{Freshness: s.reportFreshness(ctx)}
	for rows.Next() {
		var row TelegramBotBreakdownRow
		if err := rows.Scan(&row.BotID, &row.Clicks, &row.Impressions, &row.Premium); err != nil {
			return nil, err
		}
		report.Rows = append(report.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(report)
}

func (s *Service) GetTelegramPremiumReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	campaignIDs, err := s.resolveTelegramCampaignIDs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("resolve campaign filter: %w", err)
	}
	if campaignIDs != nil && len(campaignIDs) == 0 {
		return json.Marshal(TelegramPremiumReport{Freshness: s.reportFreshness(ctx)})
	}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return nil, errors.New("clickhouse connection not available")
	}
	query := `
		SELECT
			countIf(event_type = 'tg_click' AND is_premium = 1) AS premium_clicks,
			countIf(event_type = 'tg_click' AND is_premium = 0) AS non_premium_clicks
		FROM tg_events
		WHERE created_at >= ? AND created_at < ?`
	args := []any{from, to}
	var ok bool
	query, args, ok = appendTelegramCampaignFilter(query, args, campaignIDs)
	if !ok {
		return json.Marshal(TelegramPremiumReport{Freshness: s.reportFreshness(ctx)})
	}
	var premium, nonPremium int64
	if err := clickhouseQuery.QueryRow(ctx, query, args...).Scan(&premium, &nonPremium); err != nil {
		return nil, fmt.Errorf("clickhouse premium query failed: %w", err)
	}
	total := premium + nonPremium
	var rate float64
	if total > 0 {
		rate = float64(premium) * 100 / float64(total)
	}
	report := TelegramPremiumReport{
		PremiumClicks:    premium,
		NonPremiumClicks: nonPremium,
		PremiumRatePct:   rate,
		Freshness:        s.reportFreshness(ctx),
	}
	return json.Marshal(report)
}

func (s *Service) GetTelegramFraudReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	campaignIDs, err := s.resolveTelegramCampaignIDs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("resolve campaign filter: %w", err)
	}
	if campaignIDs != nil && len(campaignIDs) == 0 {
		return json.Marshal(TelegramFraudReport{Freshness: s.reportFreshness(ctx)})
	}
	clickhouseQuery := s.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return nil, errors.New("clickhouse connection not available")
	}
	query := `
		SELECT
			countIf(event_type = 'tg_click' AND JSONExtractBool(payload, 'shadow_event') = true) AS shadow_clicks,
			countIf(event_type = 'tg_click' AND JSONExtractString(payload, 'fraud_reason') != '') AS blocked_clicks
		FROM tg_events
		WHERE created_at >= ? AND created_at < ?`
	args := []any{from, to}
	var ok bool
	query, args, ok = appendTelegramCampaignFilter(query, args, campaignIDs)
	if !ok {
		return json.Marshal(TelegramFraudReport{Freshness: s.reportFreshness(ctx)})
	}
	var shadow, blocked int64
	if err := clickhouseQuery.QueryRow(ctx, query, args...).Scan(&shadow, &blocked); err != nil {
		return nil, fmt.Errorf("clickhouse fraud query failed: %w", err)
	}
	report := TelegramFraudReport{
		BlockedClicks: blocked,
		ShadowClicks:  shadow,
		Freshness:     s.reportFreshness(ctx),
	}
	return json.Marshal(report)
}

func (s *Service) GetTelegramReport(
	ctx context.Context,
	from, to time.Time,
	filter ReportFilter,
) ([]byte, error) {
	return s.GetTelegramSummaryReport(ctx, from, to, filter)
}
