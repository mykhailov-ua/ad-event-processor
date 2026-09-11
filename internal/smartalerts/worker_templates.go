package smartalerts

import (
	"context"
	"math"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func queryTemplateMetricObserved(
	ctx context.Context,
	w *Worker,
	metric string,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
	from, to time.Time,
) (float64, error) {
	template, ok := parseTemplateFromMetric(metric)
	if !ok {
		return 0, nil
	}
	switch template {
	case TemplateBudgetBurnPct:
		return queryBudgetBurnPct(ctx, w, customerID, campaignID, hasCampaign)
	case TemplateROIBelow:
		return queryAggregateROIPct(ctx, w, customerID, campaignID, hasCampaign, from, to)
	case TemplatePacingDrift:
		return queryMaxPacingDriftPct(ctx, w, customerID, campaignID, hasCampaign, from, to)
	case TemplateExportJobFailed:
		return queryFailedExportJobCount(ctx, w, customerID, from, to)
	case TemplateMarginBreach:
		return queryMarginBreachCount(ctx, w, customerID, campaignID, hasCampaign, from, to)
	default:
		return 0, nil
	}
}

func queryBudgetBurnPct(
	ctx context.Context,
	w *Worker,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
) (float64, error) {
	var campParam pgtype.UUID
	if hasCampaign {
		campParam = domain.ToUUID(campaignID)
	}
	var maxBurn float64
	err := w.host.Pool().QueryRow(ctx, `
SELECT COALESCE(MAX(
 CASE
  WHEN budget_limit > 0 THEN (current_spend::float8 / budget_limit::float8) * 100
  ELSE 0
 END
), 0)
FROM campaigns
WHERE customer_id = $1
 AND deleted_at IS NULL
 AND ($2::uuid IS NULL OR id = $2)`,
		domain.ToUUID(customerID), campParam,
	).Scan(&maxBurn)
	if err != nil {
		return 0, err
	}
	return roundAlertFloat(maxBurn), nil
}

func queryAggregateROIPct(
	ctx context.Context,
	w *Worker,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
	from, to time.Time,
) (float64, error) {
	clickhouseQuery := w.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return 0, nil
	}
	campaignIDs, err := listCampaignIDsForTemplatePG(ctx, w.host.Pool(), customerID, campaignID, hasCampaign)
	if err != nil || len(campaignIDs) == 0 {
		return 0, err
	}
	perCampaign, err := querySmartAlertMetricBatch(ctx, clickhouseQuery, "roi_pct", campaignIDs, from, to)
	if err != nil {
		return 0, err
	}
	return roundAlertFloat(aggregateAlertMetric("roi_pct", campaignIDs, perCampaign)), nil
}

func queryMaxPacingDriftPct(
	ctx context.Context,
	w *Worker,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
	from, to time.Time,
) (float64, error) {
	clickhouseQuery := w.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return 0, nil
	}
	campaignIDs, err := listCampaignIDsForTemplatePG(ctx, w.host.Pool(), customerID, campaignID, hasCampaign)
	if err != nil || len(campaignIDs) == 0 {
		return 0, err
	}
	plans, err := queryCampaignPacingPlansTemplate(ctx, w.host.Pool(), campaignIDs)
	if err != nil {
		return 0, err
	}
	spendRows, _, err := queryPacingDriftSpendRowsTemplate(ctx, clickhouseQuery, campaignIDs, from, to, len(campaignIDs)*7, 0)
	if err != nil {
		return 0, err
	}
	maxDrift := 0.0
	for _, row := range spendRows {
		plan := plans[row.campaignID]
		planned := plannedDailySpendMicroTemplate(plan.dailyBudgetMicro, row.day, plan.timezone, plan.pacingMode)
		drift := math.Abs(measuredPacingDriftPctTemplate(planned, row.actualSpendMicro) * 100)
		if drift > maxDrift {
			maxDrift = drift
		}
	}
	return roundAlertFloat(maxDrift), nil
}

func queryFailedExportJobCount(ctx context.Context, w *Worker, customerID uuid.UUID, from, to time.Time) (float64, error) {
	var count int64
	err := w.host.Pool().QueryRow(ctx, `
SELECT count(*)
FROM report_jobs
WHERE customer_id = $1
 AND status = 'FAILED'
 AND updated_at >= $2
 AND updated_at < $3`,
		domain.ToUUID(customerID), from, to,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

func queryMarginBreachCount(
	ctx context.Context,
	w *Worker,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
	from, to time.Time,
) (float64, error) {
	var campParam pgtype.UUID
	if hasCampaign {
		campParam = domain.ToUUID(campaignID)
	}
	var count int64
	err := w.host.Pool().QueryRow(ctx, `
SELECT count(*)
FROM margin_guard_activity mga
JOIN campaigns c ON c.id = mga.campaign_id
WHERE c.customer_id = $1
 AND c.deleted_at IS NULL
 AND mga.created_at >= $2
 AND mga.created_at < $3
 AND ($4::uuid IS NULL OR c.id = $4)`,
		domain.ToUUID(customerID), from, to, campParam,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

func listCampaignIDsForTemplatePG(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	hasCampaign bool,
) ([]uuid.UUID, error) {
	if pool == nil {
		return nil, nil
	}
	if hasCampaign {
		return []uuid.UUID{campaignID}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id FROM campaigns
WHERE customer_id = $1 AND deleted_at IS NULL`, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type pacingDriftSpendRowTemplate struct {
	campaignID       uuid.UUID
	day              time.Time
	actualSpendMicro int64
}

type campaignPacingPlanTemplate struct {
	dailyBudgetMicro int64
	pacingMode       string
	timezone         string
}

const (
	templatePacingDriftSpendCountQuery = `
SELECT count()
FROM (
 SELECT campaign_id, toDate(hour) AS day, sum(spend_micro) AS actual_spend_micro
 FROM placement_stats_hourly
 WHERE campaign_id IN (?)
  AND hour >= ?
  AND hour < ?
 GROUP BY campaign_id, day
)`

	templatePacingDriftSpendQuery = `
SELECT campaign_id, toDate(hour) AS day, sum(spend_micro) AS actual_spend_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id, day
LIMIT ? OFFSET ?`
)

func queryPacingDriftSpendRowsTemplate(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]pacingDriftSpendRowTemplate, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, templatePacingDriftSpendCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, templatePacingDriftSpendQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()
	out := make([]pacingDriftSpendRowTemplate, 0, limit)
	for clickhouseRows.Next() {
		var row pacingDriftSpendRowTemplate
		if err := clickhouseRows.Scan(&row.campaignID, &row.day, &row.actualSpendMicro); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}

func queryCampaignPacingPlansTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignIDs []uuid.UUID,
) (map[uuid.UUID]campaignPacingPlanTemplate, error) {
	if pool == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]campaignPacingPlanTemplate{}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, daily_budget, pacing_mode::text, COALESCE(timezone, 'UTC')
FROM campaigns
WHERE id = ANY($1::uuid[])
 AND deleted_at IS NULL`, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]campaignPacingPlanTemplate, len(campaignIDs))
	for rows.Next() {
		var id uuid.UUID
		var plan campaignPacingPlanTemplate
		if err := rows.Scan(&id, &plan.dailyBudgetMicro, &plan.pacingMode, &plan.timezone); err != nil {
			return nil, err
		}
		out[id] = plan
	}
	return out, rows.Err()
}

func plannedDailySpendMicroTemplate(dailyBudgetMicro int64, day time.Time, timezone, pacingMode string) int64 {
	if dailyBudgetMicro <= 0 {
		return 0
	}
	mode := stringsToLowerTrim(pacingMode)
	if mode == "asap" {
		return dailyBudgetMicro
	}
	loc, err := time.LoadLocation(stringsTrim(timezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}
	localDay := day.In(loc)
	endOfDay := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 23, 59, 59, 0, loc)
	now := time.Now().In(loc)
	if now.After(endOfDay) {
		return dailyBudgetMicro
	}
	startOfDay := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, loc)
	if now.Before(startOfDay) {
		return 0
	}
	daySeconds := endOfDay.Sub(startOfDay).Seconds()
	if daySeconds <= 0 {
		return dailyBudgetMicro
	}
	elapsed := now.Sub(startOfDay).Seconds()
	return int64(float64(dailyBudgetMicro) * (elapsed / daySeconds))
}

func measuredPacingDriftPctTemplate(plannedMicro, actualMicro int64) float64 {
	if plannedMicro <= 0 {
		return 0
	}
	return float64(actualMicro-plannedMicro) / float64(plannedMicro)
}

func stringsToLowerTrim(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
