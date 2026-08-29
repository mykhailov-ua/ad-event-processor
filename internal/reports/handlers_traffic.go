package reports

import (
	"context"
	"net/http"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *ReportsHTTPHandlers) registerTrafficReports(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readCampaigns := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/placements", limit(permAny(readCampaigns, h.wrapReport("placements", h.getPlacementsReport))))
	mux.HandleFunc("GET /api/v1/reports/keywords", limit(permAny(readCampaigns, h.wrapReport("keywords", h.getKeywordsReport))))
}

func (h *ReportsHTTPHandlers) getPlacementsReport(w http.ResponseWriter, r *http.Request) {
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

	limit := int32(10)
	page, err := coldpath.ParseCursorPagination(r, int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	var campaignIDs []uuid.UUID
	if campaignIDStr := r.URL.Query().Get("campaign_id"); campaignIDStr != "" {
		campaignID, err := uuid.Parse(campaignIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if !h.authorizeReportCampaign(w, r, campaignID) {
			return
		}
		campaignIDs = []uuid.UUID{campaignID}
	} else if h.Pool != nil {
		campaignIDs, err = listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, PlacementReportResponse{
			Rows:       []PlacementReportRowDTO{},
			Freshness:  h.reportFreshness(r.Context()),
			NextCursor: "",
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()

	clickhouseRows, total, err := QueryPlacementReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, int(limit), offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	ivtRates, err := QueryPlacementIVTRates(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	rows := coldpath.MapSlice(clickhouseRows, func(row reportMetricsCHRow) PlacementReportRowDTO {
		ivt := ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)]
		return ToPlacementReportRowDTO(row, ivt)
	})

	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := QueryPlacementReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, int(limit), offset)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		attachPlacementCompareDeltas(rows, prevRows)
	}

	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := PlacementReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *ReportsHTTPHandlers) getKeywordsReport(w http.ResponseWriter, r *http.Request) {
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

	limit := int32(10)
	page, err := coldpath.ParseCursorPagination(r, int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	var campaignIDs []uuid.UUID
	if campaignIDStr := r.URL.Query().Get("campaign_id"); campaignIDStr != "" {
		campaignID, parseErr := uuid.Parse(campaignIDStr)
		if parseErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if !h.authorizeReportCampaign(w, r, campaignID) {
			return
		}
		campaignIDs = []uuid.UUID{campaignID}
	} else if h.Pool != nil {
		campaignIDs, err = listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, KeywordReportResponse{
			Rows:       []KeywordReportRowDTO{},
			Freshness:  h.reportFreshness(r.Context()),
			NextCursor: "",
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()

	clickhouseRows, total, err := QueryKeywordReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, int(limit), offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	ivtRates, err := QueryKeywordIVTRates(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	rows := coldpath.MapSlice(clickhouseRows, func(row reportMetricsCHRow) KeywordReportRowDTO {
		ivt := ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)]
		return toKeywordReportRowDTO(row, ivt)
	})

	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := QueryKeywordReportRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, prevFrom, prevTo, int(limit), offset)
		if perr != nil {
			h.writeServiceError(w, perr)
			return
		}
		attachKeywordCompareDeltas(rows, prevRows)
	}

	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := KeywordReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
