package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

type ReportRowsResponse struct {
	Rows       []map[string]any `json:"rows"`
	Freshness  DataFreshnessDTO `json:"freshness"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

const spendVelocityQuery = `
SELECT
    toStartOfHour(hour) AS bucket,
    sum(spend_micro) AS spend_micro,
    sum(click_count) AS clicks
FROM placement_stats_hourly
WHERE campaign_id IN (?)
  AND hour >= ?
  AND hour < ?
GROUP BY bucket
ORDER BY bucket
LIMIT ? OFFSET ?`

const daypartHeatmapQuery = `
SELECT
    toHour(created_at) AS hour_of_day,
    count() AS clicks
FROM clicks
WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
GROUP BY hour_of_day
ORDER BY hour_of_day`

const geoDeviceQuery = `
SELECT
    coalesce(nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ') AS country,
    coalesce(
        nullIf(JSONExtractString(payload, 'device_type'), ''),
        nullIf(JSONExtractString(payload, 'device'), ''),
        'unknown'
    ) AS device,
    count() AS clicks
FROM clicks
WHERE campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
GROUP BY country, device
ORDER BY clicks DESC
LIMIT ? OFFSET ?`

const discrepancyQuery = `
SELECT
    campaign_id,
    sum(spend_micro) AS buy_micro,
    sum(revenue_micro) AS sell_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
  AND hour >= ?
  AND hour < ?
GROUP BY campaign_id
HAVING buy_micro > 0 OR sell_micro > 0
ORDER BY abs(sell_micro - buy_micro) DESC
LIMIT ? OFFSET ?`

// trueRoiQuery: Ad Spend from Cost Sync (cost_snapshots → placement_stats_hourly.spend_micro);
// revenue_micro is network-reported revenue lines when present. Conversions from tracker MV.
const trueRoiQuery = `
SELECT
    campaign_id,
    sum(spend_micro) AS ad_spend_micro,
    sum(revenue_micro) AS revenue_micro,
    sum(conversion_count) AS conversions
FROM placement_stats_hourly
WHERE campaign_id IN (?)
  AND hour >= ?
  AND hour < ?
GROUP BY campaign_id
HAVING ad_spend_micro > 0 OR revenue_micro > 0 OR conversions > 0
ORDER BY ad_spend_micro DESC
LIMIT ? OFFSET ?`

func (reports *ReportsHTTPHandlers) registerExtendedReports(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	readCampaigns := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/spend-velocity", limit(permAny(readCampaigns, reports.getSpendVelocityReport)))
	mux.HandleFunc("GET /api/v1/reports/daypart-heatmap", limit(permAny(readCampaigns, reports.getDaypartHeatmapReport)))
	mux.HandleFunc("GET /api/v1/reports/campaign-geo-device", limit(permAny(readCampaigns, reports.getCampaignGeoDeviceReport)))
	mux.HandleFunc("GET /api/v1/reports/source-quality", limit(permAny(readCampaigns, reports.getSourceQualityReport)))
	mux.HandleFunc("GET /api/v1/reports/discrepancy-buy-sell", limit(perm("customers:read", reports.getDiscrepancyBuySellReport)))
	mux.HandleFunc("GET /api/v1/reports/true-roi", limit(permAny(readCampaigns, reports.getTrueROIReport)))
	mux.HandleFunc("GET /api/v1/reports/campaign-overview", limit(permAny(readCampaigns, reports.getCampaignOverviewReport)))
	mux.HandleFunc("GET /api/v1/reports/customer-portfolio", limit(perm("customers:read", reports.getCustomerPortfolioReport)))
}

func (reports *ReportsHTTPHandlers) getSpendVelocityReport(w http.ResponseWriter, r *http.Request) {
	reports.writeCHReportRows(w, r, querySpendVelocityRows)
}

func (reports *ReportsHTTPHandlers) getTrueROIReport(w http.ResponseWriter, r *http.Request) {
	reports.writeCHReportRows(w, r, queryTrueROIRows)
}

func (reports *ReportsHTTPHandlers) getDaypartHeatmapReport(w http.ResponseWriter, r *http.Request) {
	reports.writeCHReportRows(w, r, queryDaypartHeatmapRows)
}

func (reports *ReportsHTTPHandlers) getCampaignGeoDeviceReport(w http.ResponseWriter, r *http.Request) {
	reports.writeCHReportRows(w, r, queryGeoDeviceRows)
}

func (reports *ReportsHTTPHandlers) getSourceQualityReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
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
	limit, offset := reportPageLimitOffset(r)
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{Rows: []map[string]any{}, Freshness: reports.reportFreshness(r.Context())})
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	chRows, total, err := queryPlacementReportRows(chCtx, reports.CHQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	ivtRates, err := queryPlacementIVTRates(chCtx, reports.CHQuery, campaignIDs, from, to)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(chRows))
	for _, row := range chRows {
		ivt := ivtRates[placementRowKey(row.PlacementID, row.CampaignID)]
		dto := toPlacementReportRowDTO(row, ivt)
		out = append(out, map[string]any{
			"placement_id":  dto.PlacementID,
			"campaign_id":   dto.CampaignID,
			"impressions":   dto.Impressions,
			"clicks":        dto.Clicks,
			"conversions":   dto.Conversions,
			"spend_micro":   dto.SpendMicro,
			"revenue_micro": dto.RevenueMicro,
			"roi_pct":       dto.ROIPct,
			"ctr":           dto.CTR,
			"ivt_rate":      dto.IVTRate,
		})
	}
	var nextCursor string
	if int64(offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:       out,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func (reports *ReportsHTTPHandlers) getDiscrepancyBuySellReport(w http.ResponseWriter, r *http.Request) {
	reports.writeCHReportRows(w, r, queryDiscrepancyRows)
}

func (reports *ReportsHTTPHandlers) getCampaignOverviewReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.BuyerPortfolio == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "portfolio reader not configured")
		return
	}
	portfolio, err := reports.BuyerPortfolio.GetBuyerPortfolio(r.Context(), customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	rows := make([]map[string]any, 0, len(portfolio.Campaigns))
	for _, c := range portfolio.Campaigns {
		rows = append(rows, map[string]any{
			"campaign_id":      c.ID,
			"name":             c.Name,
			"status":           c.Status,
			"impressions_7d":   c.Impressions7d,
			"clicks_7d":        c.Clicks7d,
			"utilization_pct":  c.UtilizationPct,
			"pacing_drift_pct": c.PacingDriftPct,
			"overspend_risk":   c.OverspendRisk,
		})
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:      rows,
		Freshness: reports.reportFreshness(r.Context()),
	})
}

func (reports *ReportsHTTPHandlers) getCustomerPortfolioReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.BuyerPortfolio == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "portfolio reader not configured")
		return
	}
	portfolio, err := reports.BuyerPortfolio.GetBuyerPortfolio(r.Context(), customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	rows := []map[string]any{
		{
			"active":           portfolio.Active,
			"paused":           portfolio.Paused,
			"archived":         portfolio.Archived,
			"impressions_7d":   portfolio.Impressions7d,
			"clicks_7d":        portfolio.Clicks7d,
			"overspend_count":  portfolio.OverspendCount,
			"attention_count":  len(portfolio.Attention),
			"campaigns_sample": len(portfolio.Campaigns),
		},
	}
	for _, c := range portfolio.Campaigns {
		rows = append(rows, map[string]any{
			"campaign_id":      c.ID,
			"name":             c.Name,
			"status":           c.Status,
			"impressions_7d":   c.Impressions7d,
			"clicks_7d":        c.Clicks7d,
			"utilization_pct":  c.UtilizationPct,
			"pacing_drift_pct": c.PacingDriftPct,
			"overspend_risk":   c.OverspendRisk,
			"row_type":         "campaign",
		})
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:      rows,
		Freshness: reports.reportFreshness(r.Context()),
	})
}

type chReportRowsFunc func(ctx context.Context, chQuery *database.CHQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error)

func (reports *ReportsHTTPHandlers) writeCHReportRows(w http.ResponseWriter, r *http.Request, queryFn chReportRowsFunc) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
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
	limit, offset := reportPageLimitOffset(r)
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{Rows: []map[string]any{}, Freshness: reports.reportFreshness(r.Context())})
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryFn(chCtx, reports.CHQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func reportPageLimitOffset(r *http.Request) (limit, offset int) {
	limit = 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	page, err := coldpath.Paginate(r.URL.Query().Get("cursor"), limit, 1000)
	if err != nil {
		return limit, 0
	}
	return page.Limit, page.Offset
}

func querySpendVelocityRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := chQuery.Query(ctx, spendVelocityQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("spend velocity: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var bucket time.Time
		var spendMicro, clicks int64
		if err := rows.Scan(&bucket, &spendMicro, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"bucket":      bucket.UTC().Format(time.RFC3339),
			"spend_micro": spendMicro,
			"spend":       money.FormatDecimal(spendMicro),
			"clicks":      clicks,
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func queryDaypartHeatmapRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	_, _ int,
) ([]map[string]any, int64, error) {
	rows, err := chQuery.Query(ctx, daypartHeatmapQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("daypart heatmap: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 24)
	for rows.Next() {
		var hour uint8
		var clicks uint64
		if err := rows.Scan(&hour, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"hour":   int(hour),
			"clicks": int64(clicks),
		})
	}
	return out, int64(len(out)), rows.Err()
}

func queryGeoDeviceRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := chQuery.Query(ctx, geoDeviceQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("geo device: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var country, device string
		var clicks uint64
		if err := rows.Scan(&country, &device, &clicks); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"country": country,
			"device":  device,
			"clicks":  int64(clicks),
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func queryDiscrepancyRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := chQuery.Query(ctx, discrepancyQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("discrepancy: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var campaignID uuid.UUID
		var buyMicro, sellMicro int64
		if err := rows.Scan(&campaignID, &buyMicro, &sellMicro); err != nil {
			return nil, 0, err
		}
		delta := sellMicro - buyMicro
		deltaPct := 0.0
		if buyMicro > 0 {
			deltaPct = float64(delta) / float64(buyMicro) * 100
		}
		out = append(out, map[string]any{
			"campaign_id":     campaignID.String(),
			"buy_spend_micro": buyMicro,
			"sell_rev_micro":  sellMicro,
			"delta_micro":     delta,
			"delta_pct":       deltaPct,
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}

func queryTrueROIRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := chQuery.Query(ctx, trueRoiQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("true roi: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var campaignID uuid.UUID
		var adSpendMicro, revenueMicro, conversions int64
		if err := rows.Scan(&campaignID, &adSpendMicro, &revenueMicro, &conversions); err != nil {
			return nil, 0, err
		}
		trueProfit := revenueMicro - adSpendMicro
		out = append(out, map[string]any{
			"campaign_id":       campaignID.String(),
			"ad_spend_micro":    adSpendMicro,
			"revenue_micro":     revenueMicro,
			"true_profit_micro": trueProfit,
			"true_roi_pct":      calcROIPct(trueProfit, adSpendMicro),
			"true_cpa_micro":    calcCPAMicro(adSpendMicro, conversions),
			"conversions":       conversions,
		})
	}
	return out, int64(len(out) + offset), rows.Err()
}
