package reports

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type WireSignalBreakdownRowDTO struct {
	CampaignID         string  `json:"campaign_id"`
	FraudReason        string  `json:"fraud_reason,omitempty"`
	FraudCategory      string  `json:"fraud_category,omitempty"`
	FraudCategoryLabel string  `json:"fraud_category_label,omitempty"`
	EventCount         int64   `json:"event_count"`
	SilentRejectCount  int64   `json:"silent_reject_count"`
	SilentRejectRatio  float64 `json:"silent_reject_ratio"`
	SignalsDegraded    bool    `json:"signals_degraded,omitempty"`
}

type WireSignalBreakdownReportResponse struct {
	Rows       []WireSignalBreakdownRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO            `json:"freshness"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}

func (h *ReportsHTTPHandlers) registerWireSignalBreakdownReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/wire-signal-breakdown", limit(permAny(ReportPermsFraudCustomer(), h.wrapReport("wire-signal-breakdown", h.getWireSignalBreakdownReport))))
}

func (h *ReportsHTTPHandlers) getWireSignalBreakdownReport(w http.ResponseWriter, r *http.Request) {
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
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, WireSignalBreakdownReportResponse{
			Rows:      []WireSignalBreakdownRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	rows, total, err := queryWireSignalBreakdownRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset, r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, WireSignalBreakdownReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
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
) ([]WireSignalBreakdownRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rawRows, _, err := queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, 10_000, 0)
	if err != nil {
		return nil, 0, err
	}
	return filterWireSignalBreakdownRows(rawRows, limit, offset, scrubCtx)
}

func filterWireSignalBreakdownRows(raw []FraudBreakdownRowDTO, limit, offset int, scrubCtx context.Context) ([]WireSignalBreakdownRowDTO, int64, error) {
	filtered := make([]WireSignalBreakdownRowDTO, 0, len(raw))
	for _, row := range raw {
		if !isWireSignalReason(row.FraudReason) {
			continue
		}
		out := WireSignalBreakdownRowDTO{
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
