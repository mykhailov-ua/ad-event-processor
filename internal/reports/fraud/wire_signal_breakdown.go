package fraud

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func registerWireSignalBreakdownReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/wire-signal-breakdown", limit(permAny(ReportPermsFraudCustomer(), h.WrapReport("wire-signal-breakdown", func(w http.ResponseWriter, r *http.Request) { getWireSignalBreakdownReport(h, w, r) }))))
}

func getWireSignalBreakdownReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.ResolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := reports.ParseReportRange(r)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, reports.WireSignalBreakdownReportResponse{
			Rows:      []reports.WireSignalBreakdownRowDTO{},
			Freshness: h.ReportFreshness(r.Context()),
		})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, total, err := queryWireSignalBreakdownRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset, r.Context())
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.WireSignalBreakdownReportResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryWireSignalBreakdownRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
	scrubCtx context.Context,
) ([]reports.WireSignalBreakdownRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rawRows, _, err := queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, 10_000, 0)
	if err != nil {
		return nil, 0, err
	}
	return filterWireSignalBreakdownRows(rawRows, limit, offset, scrubCtx)
}

func filterWireSignalBreakdownRows(raw []reports.FraudBreakdownRowDTO, limit, offset int, scrubCtx context.Context) ([]reports.WireSignalBreakdownRowDTO, int64, error) {
	filtered := make([]reports.WireSignalBreakdownRowDTO, 0, len(raw))
	for _, row := range raw {
		if !isWireSignalReason(row.FraudReason) {
			continue
		}
		out := reports.WireSignalBreakdownRowDTO{
			CampaignID:        row.CampaignID,
			EventCount:        row.EventCount,
			SilentRejectCount: row.SilentRejectCount,
		}
		if out.EventCount > 0 {
			out.SilentRejectRatio = calcSilentRejectRatio(out.SilentRejectCount, out.EventCount)
		}
		if maskLevelFromContext(scrubCtx) == authz.MaskFull {
			out.FraudReason = row.FraudReason
		} else {
			category, label := FraudReasonToCategory(row.FraudReason)
			out.FraudCategory = category
			out.FraudCategoryLabel = label
		}
		filtered = append(filtered, out)
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}
