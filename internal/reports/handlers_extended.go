package reports

import (
	"context"
	"errors"
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
	fetch, ok := h.fetchClickHouseReportRows(w, r, querySpendVelocityRows)
	if !ok {
		return
	}
	out := spendVelocityRowsFromMaps(fetch.Rows)
	if parseComparePrevious(r) {
		prevRows, perr := h.loadClickHouseReportRowsPrevious(r, querySpendVelocityRows)
		if perr != nil {
			h.writeLoadedReportError(w, perr)
			return
		}
		attachSpendVelocityCompareDeltas(out, spendVelocityRowsFromMaps(prevRows))
	}
	httpresponse.JSON(w, http.StatusOK, SpendVelocityReportResponse{
		Rows:       out,
		Freshness:  fetch.Freshness,
		NextCursor: fetch.NextCursor,
	})
}

func (h *ReportsHTTPHandlers) getTrueROIReport(w http.ResponseWriter, r *http.Request) {
	fetch, ok := h.fetchClickHouseReportRows(w, r, queryTrueROIRows)
	if !ok {
		return
	}
	out := trueROIRowsFromMaps(fetch.Rows)
	if parseComparePrevious(r) {
		prevRows, perr := h.loadClickHouseReportRowsPrevious(r, queryTrueROIRows)
		if perr != nil {
			h.writeLoadedReportError(w, perr)
			return
		}
		attachTrueROICompareDeltas(out, trueROIRowsFromMaps(prevRows))
	}
	httpresponse.JSON(w, http.StatusOK, TrueROIReportResponse{
		Rows:       out,
		Freshness:  fetch.Freshness,
		NextCursor: fetch.NextCursor,
	})
}

func (h *ReportsHTTPHandlers) getDaypartHeatmapReport(w http.ResponseWriter, r *http.Request) {
	fetch, ok := h.fetchClickHouseReportRows(w, r, queryDaypartHeatmapRows)
	if !ok {
		return
	}
	out := daypartHeatmapRowsFromMaps(fetch.Rows)
	if parseComparePrevious(r) {
		prevRows, perr := h.loadClickHouseReportRowsPrevious(r, queryDaypartHeatmapRows)
		if perr != nil {
			h.writeLoadedReportError(w, perr)
			return
		}
		attachDaypartHeatmapCompareDeltas(out, daypartHeatmapRowsFromMaps(prevRows))
	}
	httpresponse.JSON(w, http.StatusOK, DaypartHeatmapReportResponse{
		Rows:       out,
		Freshness:  fetch.Freshness,
		NextCursor: fetch.NextCursor,
	})
}

func (h *ReportsHTTPHandlers) getCampaignGeoDeviceReport(w http.ResponseWriter, r *http.Request) {
	fetch, ok := h.fetchClickHouseReportRows(w, r, queryGeoDeviceRows)
	if !ok {
		return
	}
	httpresponse.JSON(w, http.StatusOK, CampaignGeoDeviceReportResponse{
		Rows:       campaignGeoDeviceRowsFromMaps(fetch.Rows),
		Freshness:  fetch.Freshness,
		NextCursor: fetch.NextCursor,
	})
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
		httpresponse.JSON(w, http.StatusOK, SourceQualityReportResponse{
			Rows:      []SourceQualityRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
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
			attachSourceQualityCompareDeltas(out, prevOut)
		}
		var nextCursor string
		if int64(offset)+int64(len(out)) < total {
			nextCursor = coldpath.EncodeCursor(offset + limit)
		}
		httpresponse.JSON(w, http.StatusOK, SourceQualityReportResponse{
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
	out := make([]SourceQualityRowDTO, 0, len(clickhouseRows))
	for _, row := range clickhouseRows {
		dto := ToPlacementReportRowDTO(row, ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)])
		out = append(out, sourceQualityRowFromPlacement(dto))
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
		prevOut := make([]SourceQualityRowDTO, 0, len(prevRows))
		for _, row := range prevRows {
			dto := ToPlacementReportRowDTO(row, prevIVT[ReportMetricsKey(row.Dimension, row.CampaignID)])
			prevOut = append(prevOut, sourceQualityRowFromPlacement(dto))
		}
		attachSourceQualityCompareDeltas(out, prevOut)
	}
	var nextCursor string
	if int64(offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	httpresponse.JSON(w, http.StatusOK, SourceQualityReportResponse{
		Rows:       out,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func (h *ReportsHTTPHandlers) getDiscrepancyBuySellReport(w http.ResponseWriter, r *http.Request) {
	fetch, ok := h.fetchClickHouseReportRows(w, r, queryDiscrepancyRows)
	if !ok {
		return
	}
	httpresponse.JSON(w, http.StatusOK, DiscrepancyBuySellReportResponse{
		Rows:       discrepancyBuySellRowsFromMaps(fetch.Rows),
		Freshness:  fetch.Freshness,
		NextCursor: fetch.NextCursor,
	})
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
	httpresponse.JSON(w, http.StatusOK, CampaignOverviewReportResponse{
		Rows:      campaignOverviewRowsFromPortfolio(portfolio.Campaigns),
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
	httpresponse.JSON(w, http.StatusOK, CustomerPortfolioReportResponse{
		Summary:   customerPortfolioSummaryFromDTO(portfolio),
		Campaigns: customerPortfolioCampaignRowsFromDTO(portfolio.Campaigns),
		Freshness: h.reportFreshness(r.Context()),
	})
}

type clickhouseReportRowsFunc func(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error)

type clickhouseReportFetch struct {
	Rows       []map[string]any
	NextCursor string
	Freshness  DataFreshnessDTO
}

func (h *ReportsHTTPHandlers) loadClickHouseReportRows(
	r *http.Request,
	queryFn clickhouseReportRowsFunc,
) (clickhouseReportFetch, error) {
	customerID, err := h.parseReportCustomerID(r)
	if err != nil {
		return clickhouseReportFetch{}, err
	}
	if h.ClickHouseQuery == nil {
		return clickhouseReportFetch{}, errClickHouseUnavailable
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		return clickhouseReportFetch{}, err
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		return clickhouseReportFetch{}, errInvalidReportCursor
	}
	limit, offset := page.Limit, page.Offset
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		return clickhouseReportFetch{}, err
	}
	freshness := h.reportFreshness(r.Context())
	if len(campaignIDs) == 0 {
		return clickhouseReportFetch{Freshness: freshness}, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryFn(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return clickhouseReportFetch{}, err
	}
	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + limit)
	}
	return clickhouseReportFetch{
		Rows:       rows,
		NextCursor: nextCursor,
		Freshness:  freshness,
	}, nil
}

func (h *ReportsHTTPHandlers) loadClickHouseReportRowsPrevious(
	r *http.Request,
	queryFn clickhouseReportRowsFunc,
) ([]map[string]any, error) {
	customerID, err := h.parseReportCustomerID(r)
	if err != nil {
		return nil, err
	}
	if h.ClickHouseQuery == nil {
		return nil, errClickHouseUnavailable
	}
	from, to, err := ParseReportRange(r)
	if err != nil {
		return nil, err
	}
	prevFrom, prevTo := previousReportRange(from, to)
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		return nil, errInvalidReportCursor
	}
	limit, offset := page.Limit, page.Offset
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return nil, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, _, err := queryFn(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, limit, offset)
	return rows, err
}

func (h *ReportsHTTPHandlers) fetchClickHouseReportRows(
	w http.ResponseWriter,
	r *http.Request,
	queryFn clickhouseReportRowsFunc,
) (*clickhouseReportFetch, bool) {
	if _, ok := h.resolveReportCustomerID(w, r); !ok {
		return nil, false
	}
	fetch, err := h.loadClickHouseReportRows(r, queryFn)
	if err != nil {
		h.writeLoadedReportError(w, err)
		return nil, false
	}
	return &fetch, true
}

func (h *ReportsHTTPHandlers) writeLoadedReportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errClickHouseUnavailable):
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
	case errors.Is(err, errInvalidReportCursor):
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
	case errors.Is(err, errInvalidReportCustomerID), errors.Is(err, errReportCustomerIDRequired):
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
	default:
		h.writeServiceError(w, err)
	}
}
