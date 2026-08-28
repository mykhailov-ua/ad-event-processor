package reports

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

type ReportRowsResponse struct {
	Rows       []map[string]any `json:"rows"`
	Freshness  DataFreshnessDTO `json:"freshness"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type TrueROIReportRow struct {
	CampaignID      string  `json:"campaign_id"`
	AdSpendMicro    int64   `json:"ad_spend_micro"`
	RevenueMicro    int64   `json:"revenue_micro"`
	TrueProfitMicro int64   `json:"true_profit_micro"`
	TrueRoiPct      float64 `json:"true_roi_pct"`
	TrueCpaMicro    int64   `json:"true_cpa_micro"`
	Conversions     int64   `json:"conversions"`
}

func trueROIReportRowsToMaps(rows []TrueROIReportRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"campaign_id":       r.CampaignID,
			"ad_spend_micro":    r.AdSpendMicro,
			"revenue_micro":     r.RevenueMicro,
			"true_profit_micro": r.TrueProfitMicro,
			"true_roi_pct":      r.TrueRoiPct,
			"true_cpa_micro":    r.TrueCpaMicro,
			"conversions":       r.Conversions,
		})
	}
	return out
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
 coalesce(` + clickhouseDimCountryExpr + `, 'ZZ') AS country,
 coalesce(
 ` + clickhouseDimDeviceExpr + `
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

func (h *ReportsHTTPHandlers) registerExtendedReports(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	readCampaigns := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/spend-velocity", limit(permAny(readCampaigns, h.wrapReport("spend-velocity", h.getSpendVelocityReport))))
	mux.HandleFunc("GET /api/v1/reports/daypart-heatmap", limit(permAny(readCampaigns, h.wrapReport("daypart-heatmap", h.getDaypartHeatmapReport))))
	mux.HandleFunc("GET /api/v1/reports/campaign-geo-device", limit(permAny(readCampaigns, h.wrapReport("campaign-geo-device", h.getCampaignGeoDeviceReport))))
	mux.HandleFunc("GET /api/v1/reports/source-quality", limit(permAny(readCampaigns, h.wrapReport("source-quality", h.getSourceQualityReport))))
	mux.HandleFunc("GET /api/v1/reports/discrepancy-buy-sell", limit(perm("customers:read", h.wrapReport("discrepancy-buy-sell", h.getDiscrepancyBuySellReport))))
	mux.HandleFunc("GET /api/v1/reports/true-roi", limit(permAny(readCampaigns, h.wrapReport("true-roi", h.getTrueROIReport))))
	mux.HandleFunc("GET /api/v1/reports/campaign-overview", limit(permAny(readCampaigns, h.wrapReport("campaign-overview", h.getCampaignOverviewReport))))
	mux.HandleFunc("GET /api/v1/reports/customer-portfolio", limit(perm("customers:read", h.wrapReport("customer-portfolio", h.getCustomerPortfolioReport))))
}

func (h *ReportsHTTPHandlers) getSpendVelocityReport(w http.ResponseWriter, r *http.Request) {
	h.writeClickHouseReportRows(w, r, querySpendVelocityRows, nil)
}

func (h *ReportsHTTPHandlers) getTrueROIReport(w http.ResponseWriter, r *http.Request) {
	h.writeClickHouseReportRows(w, r, queryTrueROIRows, []string{"campaign_id"})
}

func (h *ReportsHTTPHandlers) getDaypartHeatmapReport(w http.ResponseWriter, r *http.Request) {
	h.writeClickHouseReportRows(w, r, queryDaypartHeatmapRows, []string{"hour"})
}

func (h *ReportsHTTPHandlers) getCampaignGeoDeviceReport(w http.ResponseWriter, r *http.Request) {
	h.writeClickHouseReportRows(w, r, queryGeoDeviceRows, nil)
}

func (h *ReportsHTTPHandlers) getSourceQualityReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	limit, offset := page.Limit, page.Offset
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{Rows: []map[string]any{}, Freshness: h.reportFreshness(r.Context())})
		return
	}
	groupBy := parseSourceQualityGroupBy(r)
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()

	if sourceQualityNeedsDetailRows(groupBy) {
		out, total, err := querySourceQualityDetailRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, limit, offset)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		if parseComparePrevious(r) {
			prevFrom, prevTo := previousReportRange(from, to)
			prevOut, _, perr := querySourceQualityDetailRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, limit, offset)
			if perr != nil {
				h.writeServiceError(w, perr)
				return
			}
			attachSourceQualityDetailCompareDeltas(out, prevOut)
		}
		var nextCursor string
		if int64(offset)+int64(len(out)) < total {
			nextCursor = coldpath.EncodeCursor(offset + limit)
		}
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
			Rows:       out,
			Freshness:  h.reportFreshness(r.Context()),
			NextCursor: nextCursor,
		})
		return
	}

	clickhouseRows, total, err := QueryPlacementReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	ivtRates, err := QueryPlacementIVTRates(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(clickhouseRows))
	for _, row := range clickhouseRows {
		ivt := ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)]
		dto := ToPlacementReportRowDTO(row, ivt)
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
	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := QueryPlacementReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, limit, offset)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		prevIVT, perr := QueryPlacementIVTRates(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		prevOut := make([]map[string]any, 0, len(prevRows))
		for _, row := range prevRows {
			ivt := prevIVT[ReportMetricsKey(row.Dimension, row.CampaignID)]
			dto := ToPlacementReportRowDTO(row, ivt)
			prevOut = append(prevOut, map[string]any{
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
		attachMapCompareDeltas(out, prevOut, "placement_id", "campaign_id")
	}
	var nextCursor string
	if int64(offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:       out,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func (h *ReportsHTTPHandlers) getDiscrepancyBuySellReport(w http.ResponseWriter, r *http.Request) {
	h.writeClickHouseReportRows(w, r, queryDiscrepancyRows, nil)
}

func (h *ReportsHTTPHandlers) getCampaignOverviewReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.BuyerPortfolio == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "portfolio reader not configured")
		return
	}
	portfolio, err := h.BuyerPortfolio.GetBuyerPortfolio(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
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
		Freshness: h.reportFreshness(r.Context()),
	})
}

func (h *ReportsHTTPHandlers) getCustomerPortfolioReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.BuyerPortfolio == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "portfolio reader not configured")
		return
	}
	portfolio, err := h.BuyerPortfolio.GetBuyerPortfolio(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
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
		Freshness: h.reportFreshness(r.Context()),
	})
}

type clickhouseReportRowsFunc func(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error)

func (h *ReportsHTTPHandlers) writeClickHouseReportRows(
	w http.ResponseWriter,
	r *http.Request,
	queryFn clickhouseReportRowsFunc,
	compareKeyFields []string,
) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	limit, offset := page.Limit, page.Offset
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{Rows: []map[string]any{}, Freshness: h.reportFreshness(r.Context())})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryFn(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if parseComparePrevious(r) && len(compareKeyFields) > 0 {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := queryFn(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, limit, offset)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		attachMapCompareDeltas(rows, prevRows, compareKeyFields...)
	}
	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func querySpendVelocityRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, spendVelocityQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("spend velocity: %w", err)
	}
	defer func() { _ = rows.Close() }()
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
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	_, _ int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, daypartHeatmapQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("daypart heatmap: %w", err)
	}
	defer func() { _ = rows.Close() }()
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
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, geoDeviceQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("geo device: %w", err)
	}
	defer func() { _ = rows.Close() }()
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
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, discrepancyQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("discrepancy: %w", err)
	}
	defer func() { _ = rows.Close() }()
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
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	rows, err := clickhouseQuery.Query(ctx, trueRoiQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("true roi: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]TrueROIReportRow, 0, limit)
	for rows.Next() {
		var campaignID uuid.UUID
		var adSpendMicro, revenueMicro, conversions int64
		if err := rows.Scan(&campaignID, &adSpendMicro, &revenueMicro, &conversions); err != nil {
			return nil, 0, err
		}
		trueProfit := revenueMicro - adSpendMicro
		out = append(out, TrueROIReportRow{
			CampaignID:      campaignID.String(),
			AdSpendMicro:    adSpendMicro,
			RevenueMicro:    revenueMicro,
			TrueProfitMicro: trueProfit,
			TrueRoiPct:      calcROIPct(trueProfit, adSpendMicro),
			TrueCpaMicro:    calcCPAMicro(adSpendMicro, conversions),
			Conversions:     conversions,
		})
	}
	return trueROIReportRowsToMaps(out), int64(len(out) + offset), rows.Err()
}
