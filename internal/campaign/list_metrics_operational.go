package campaign

import (
	"context"
	"math"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const campaignListPacingDriftThreshold = 0.15

var operationalNow = time.Now

type campaignOperationalMeta struct {
	budgetLimit     int64
	currentSpend    int64
	pacingMode      string
	status          string
	dailyBudget     int64
	timezone        string
	todaySpendMicro int64
}

const campaignTodaySpendQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 sum(spend_micro) AS spend_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id`

func attachCampaignListOperationalSignals(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	items map[string]CampaignListMetricsRowDTO,
	batchStale bool,
	metricsAsOf string,
) error {
	if pool == nil || len(items) == 0 {
		return nil
	}
	metaByID, err := loadCampaignOperationalMeta(ctx, pool, campaignIDs)
	if err != nil {
		return err
	}
	if clickhouseQuery != nil && len(campaignIDs) > 0 {
		chCtx, cancel := context.WithTimeout(ctx, campaignListMetricsCHTimeout)
		defer cancel()
		todaySpend, spendErr := queryCampaignTodaySpendMicro(chCtx, clickhouseQuery, metaByID)
		if spendErr == nil {
			for id, spend := range todaySpend {
				if meta, ok := metaByID[id]; ok {
					meta.todaySpendMicro = spend
					metaByID[id] = meta
				}
			}
		}
	}
	if metricsAsOf == "" {
		metricsAsOf = time.Now().UTC().Format(time.RFC3339)
	}
	for id, entry := range items {
		meta, ok := metaByID[id]
		if !ok {
			entry.MetricsStale = batchStale
			entry.MetricsAsOf = metricsAsOf
			items[id] = entry
			continue
		}
		entry.BudgetBurnPct = campaignBudgetBurnPct(meta.currentSpend, meta.budgetLimit)
		entry.PacingMode = meta.pacingMode
		entry.PacingHealth = deriveCampaignPacingHealth(meta)
		entry.MetricsStale = batchStale
		entry.MetricsAsOf = metricsAsOf
		items[id] = entry
	}
	return nil
}

func campaignBudgetBurnPct(currentSpend, budgetLimit int64) float64 {
	if budgetLimit <= 0 {
		return 0
	}
	return float64(currentSpend) / float64(budgetLimit) * 100
}

func deriveCampaignPacingHealth(meta campaignOperationalMeta) string {
	if strings.EqualFold(meta.status, "EXHAUSTED") {
		return "exhausted"
	}
	burn := campaignBudgetBurnPct(meta.currentSpend, meta.budgetLimit)
	if burn >= 100 {
		return "exhausted"
	}
	mode := strings.ToUpper(strings.TrimSpace(meta.pacingMode))
	if mode != "EVEN" {
		return "ok"
	}
	drift := operationalPacingDriftToday(meta)
	if math.Abs(drift) >= campaignListPacingDriftThreshold {
		return "drift"
	}
	return "ok"
}

func operationalPacingDriftToday(meta campaignOperationalMeta) float64 {
	if meta.dailyBudget <= 0 {
		return 0
	}
	loc, err := time.LoadLocation(strings.TrimSpace(meta.timezone))
	if err != nil || loc == nil {
		loc = time.UTC
	}
	now := operationalNow().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	planned := plannedOperationalDailySpendMicro(meta.dailyBudget, dayStart, meta.timezone, meta.pacingMode, now)
	if planned <= 0 {
		return 0
	}
	return float64(meta.todaySpendMicro-planned) / float64(planned)
}

func plannedOperationalDailySpendMicro(dailyBudgetMicro int64, day time.Time, timezone, pacingMode string, now time.Time) int64 {
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
	localNow := now.In(loc)
	if !localDay.Truncate(24 * time.Hour).Equal(localNow.Truncate(24 * time.Hour)) {
		return dailyBudgetMicro
	}
	ratio := operationalPacingExpectedRatio(localNow)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	return int64(float64(dailyBudgetMicro) * ratio)
}

func operationalPacingExpectedRatio(localNow time.Time) float64 {
	currentHour := localNow.Hour()
	minuteFrac := (float64(localNow.Minute()) + float64(localNow.Second())/60.0) / 60.0
	var elapsedWeight float64
	for h := range 24 {
		w := 1.0 / 24.0
		switch {
		case h < currentHour:
			elapsedWeight += w
		case h == currentHour:
			elapsedWeight += w * minuteFrac
		}
	}
	ratio := elapsedWeight
	if ratio > 1 {
		return 1
	}
	return ratio
}

func loadCampaignOperationalMeta(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignIDs []uuid.UUID,
) (map[string]campaignOperationalMeta, error) {
	out := make(map[string]campaignOperationalMeta, len(campaignIDs))
	if len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, budget_limit, current_spend, pacing_mode::text, status, daily_budget, COALESCE(timezone, 'UTC')
		FROM campaigns
		WHERE id = ANY($1::uuid[])
		  AND deleted_at IS NULL`, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var meta campaignOperationalMeta
		if err := rows.Scan(&id, &meta.budgetLimit, &meta.currentSpend, &meta.pacingMode, &meta.status, &meta.dailyBudget, &meta.timezone); err != nil {
			return nil, err
		}
		out[id.String()] = meta
	}
	return out, rows.Err()
}

func queryCampaignTodaySpendMicro(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	metaByID map[string]campaignOperationalMeta,
) (map[string]int64, error) {
	out := make(map[string]int64)
	if clickhouseQuery == nil || len(metaByID) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(metaByID))
	type window struct {
		from time.Time
		to   time.Time
	}
	windows := make(map[string]window, len(metaByID))
	now := operationalNow().UTC()
	for campaignID, meta := range metaByID {
		id, err := uuid.Parse(campaignID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		loc, err := time.LoadLocation(strings.TrimSpace(meta.timezone))
		if err != nil || loc == nil {
			loc = time.UTC
		}
		localNow := now.In(loc)
		start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
		windows[campaignID] = window{from: start.UTC(), to: now}
	}
	if len(ids) == 0 {
		return out, nil
	}
	// Single UTC-day query covers most campaigns; per-timezone refinement uses local midnight mapped to UTC above.
	minFrom := now
	maxTo := time.Time{}
	for _, w := range windows {
		if w.from.Before(minFrom) {
			minFrom = w.from
		}
		if w.to.After(maxTo) {
			maxTo = w.to
		}
	}
	rows, err := clickhouseQuery.Query(ctx, campaignTodaySpendQuery, ids, minFrom, maxTo)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var campaignID string
		var spend int64
		if err := rows.Scan(&campaignID, &spend); err != nil {
			return nil, err
		}
		out[campaignID] = spend
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func campaignListMetricsAsOf(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, batchStale bool) (string, bool) {
	freshness := reports.DataFreshnessFromClickHouse(ctx, clickhouseQuery)
	stale := batchStale || freshness.Stale
	asOf := freshness.AsOf
	if asOf == "" {
		asOf = time.Now().UTC().Format(time.RFC3339)
	}
	return asOf, stale
}
