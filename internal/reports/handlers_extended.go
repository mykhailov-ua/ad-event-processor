package reports

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type ReportRowsResponse struct {
	Rows       []map[string]any `json:"rows"`
	Freshness  DataFreshnessDTO `json:"freshness"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

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
		httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(nil, h.reportFreshness(r.Context()), ""))
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
		httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(out, h.reportFreshness(r.Context()), nextCursor))
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
	httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(out, h.reportFreshness(r.Context()), nextCursor))
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
	httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(rows, h.reportFreshness(r.Context()), ""))
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
	httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(rows, h.reportFreshness(r.Context()), ""))
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
		httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(nil, h.reportFreshness(r.Context()), ""))
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
	httpresponse.JSON(w, http.StatusOK, NewReportRowsResponse(rows, h.reportFreshness(r.Context()), nextCursor))
}
