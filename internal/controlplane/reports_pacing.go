package controlplane

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PacingDriftRowDTO struct {
	CampaignID        string  `json:"campaign_id"`
	Date              string  `json:"date"`
	PlannedSpendMicro int64   `json:"planned_spend_micro"`
	ActualSpendMicro  int64   `json:"actual_spend_micro"`
	DriftPct          float64 `json:"drift_pct"`
	PacingMode        string  `json:"pacing_mode,omitempty"`
}

type PacingDriftReportResponse struct {
	Rows       []PacingDriftRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO    `json:"freshness"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

const pacingDriftSpendQuery = `
SELECT
 campaign_id,
 toDate(hour) AS day,
 sum(spend_micro) AS actual_spend_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id, day
ORDER BY day DESC, campaign_id
LIMIT ? OFFSET ?`

const pacingDriftSpendCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, toDate(hour) AS day
 FROM placement_stats_hourly
 WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
 GROUP BY campaign_id, day
)`

func (reports *ReportsHTTPHandlers) registerPacingDriftReport(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readCampaigns := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/pacing-drift", limit(permAny(readCampaigns, reports.wrapReport("pacing-drift", reports.getPacingDriftReport))))
}

func (reports *ReportsHTTPHandlers) getPacingDriftReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.Pool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "postgres not configured")
		return
	}
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, PacingDriftReportResponse{
			Rows:      []PacingDriftRowDTO{},
			Freshness: reports.reportFreshness(r.Context()),
		})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	spendRows, total, err := queryPacingDriftSpendRows(chCtx, reports.CHQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	pgCtx, pgCancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer pgCancel()
	plans, err := queryCampaignPacingPlans(pgCtx, reports.Pool, campaignIDs)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	out := make([]PacingDriftRowDTO, 0, len(spendRows))
	for _, row := range spendRows {
		plan := plans[row.campaignID]
		planned := plannedDailySpendMicro(plan.dailyBudgetMicro, row.day, plan.timezone, plan.pacingMode)
		drift := measuredPacingDriftPct(planned, row.actualSpendMicro)
		out = append(out, PacingDriftRowDTO{
			CampaignID:        row.campaignID.String(),
			Date:              row.day.UTC().Format("2006-01-02"),
			PlannedSpendMicro: planned,
			ActualSpendMicro:  row.actualSpendMicro,
			DriftPct:          drift,
			PacingMode:        plan.pacingMode,
		})
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, PacingDriftReportResponse{
		Rows:       out,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

type pacingDriftSpendRow struct {
	campaignID       uuid.UUID
	day              time.Time
	actualSpendMicro int64
}

type campaignPacingPlan struct {
	dailyBudgetMicro int64
	pacingMode       string
	timezone         string
}

func queryPacingDriftSpendRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]pacingDriftSpendRow, int64, error) {
	if chQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, pacingDriftSpendCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, pacingDriftSpendQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	out := make([]pacingDriftSpendRow, 0, limit)
	for chRows.Next() {
		var row pacingDriftSpendRow
		if err := chRows.Scan(&row.campaignID, &row.day, &row.actualSpendMicro); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func queryCampaignPacingPlans(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignIDs []uuid.UUID,
) (map[uuid.UUID]campaignPacingPlan, error) {
	if pool == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]campaignPacingPlan{}, nil
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

	out := make(map[uuid.UUID]campaignPacingPlan, len(campaignIDs))
	for rows.Next() {
		var id uuid.UUID
		var plan campaignPacingPlan
		if err := rows.Scan(&id, &plan.dailyBudgetMicro, &plan.pacingMode, &plan.timezone); err != nil {
			return nil, err
		}
		out[id] = plan
	}
	return out, rows.Err()
}

func plannedDailySpendMicro(dailyBudgetMicro int64, day time.Time, timezone, pacingMode string) int64 {
	if dailyBudgetMicro <= 0 {
		return 0
	}
	mode := strings.ToLower(strings.TrimSpace(pacingMode))
	if mode == "asap" {
		return dailyBudgetMicro
	}
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}
	localDay := day.In(loc)
	endOfDay := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 23, 59, 59, 0, loc)
	now := time.Now().In(loc)
	if !localDay.Truncate(24 * time.Hour).Equal(now.Truncate(24 * time.Hour)) {
		return dailyBudgetMicro
	}
	ratio := smartPacingExpectedRatio(uniformHourWeights(), nil, now)
	if endOfDay.Before(now) {
		ratio = 1
	}
	return int64(float64(dailyBudgetMicro) * ratio)
}

func measuredPacingDriftPct(plannedMicro, actualMicro int64) float64 {
	if plannedMicro <= 0 {
		return 0
	}
	return (float64(actualMicro) - float64(plannedMicro)) / float64(plannedMicro)
}

func queryPacingDriftExportRows(
	ctx context.Context,
	deps ReportExportDeps,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]PacingDriftRowDTO, int64, error) {
	if deps.Pool == nil || deps.CHQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	spendRows, total, err := queryPacingDriftSpendRows(ctx, deps.CHQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	plans, err := queryCampaignPacingPlans(ctx, deps.Pool, campaignIDs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PacingDriftRowDTO, 0, len(spendRows))
	for _, row := range spendRows {
		plan := plans[row.campaignID]
		planned := plannedDailySpendMicro(plan.dailyBudgetMicro, row.day, plan.timezone, plan.pacingMode)
		out = append(out, PacingDriftRowDTO{
			CampaignID:        row.campaignID.String(),
			Date:              row.day.UTC().Format("2006-01-02"),
			PlannedSpendMicro: planned,
			ActualSpendMicro:  row.actualSpendMicro,
			DriftPct:          measuredPacingDriftPct(planned, row.actualSpendMicro),
			PacingMode:        plan.pacingMode,
		})
	}
	return out, total, nil
}
