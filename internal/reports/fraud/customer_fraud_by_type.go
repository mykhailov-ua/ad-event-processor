package fraud

import (
	"context"
	"net/http"
	"sort"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func registerCustomerFraudByTypeReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/customer-fraud-by-type", limit(permAny(ReportPermsFraudCustomer(), h.WrapReport("customer-fraud-by-type", func(w http.ResponseWriter, r *http.Request) { getCustomerFraudByTypeReport(h, w, r) }))))
}

func getCustomerFraudByTypeReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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
	campaignFilter := r.URL.Query().Get("campaign_id")
	categoryFilter := r.URL.Query().Get("fraud_category")

	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if campaignFilter != "" {
		filterID, parseErr := uuid.Parse(campaignFilter)
		if parseErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		allowed := false
		for _, id := range campaignIDs {
			if id == filterID {
				allowed = true
				break
			}
		}
		if !allowed {
			reports.WriteReportsHandlerError(h, w, reports.ErrForbidden)
			return
		}
		campaignIDs = []uuid.UUID{filterID}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, reports.CustomerFraudByTypeReportResponse{
			Rows:      []reports.CustomerFraudByTypeRowDTO{},
			Freshness: h.ReportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rawRows, _, err := queryFraudBreakdownRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, 10_000, 0)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	rows := aggregateCustomerFraudByType(rawRows, categoryFilter)
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	total := int64(len(rows))
	if page.Offset >= len(rows) {
		rows = nil
	} else {
		end := page.Offset + page.Limit
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[page.Offset:end]
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.CustomerFraudByTypeReportResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func aggregateCustomerFraudByType(rows []reports.FraudBreakdownRowDTO, categoryFilter string) []reports.CustomerFraudByTypeRowDTO {
	type agg struct {
		events        int64
		silentRejects int64
	}
	byKey := make(map[reports.FraudCategoryAggKey]*agg)
	totalsByCampaign := make(map[string]int64)
	for _, row := range rows {
		categories := FraudReasonCategoriesFromField(row.FraudReason)
		if len(categories) == 0 {
			categories = []string{fraudCategoryOther}
		}
		share := int64(1)
		if len(categories) > 1 {
			share = 1
		}
		for _, category := range categories {
			if categoryFilter != "" && category != categoryFilter {
				continue
			}
			key := reports.FraudCategoryAggKey{CampaignID: row.CampaignID, Category: category}
			slot := byKey[key]
			if slot == nil {
				slot = &agg{}
				byKey[key] = slot
			}
			slot.events += row.EventCount / int64(len(categories))
			if share == 1 {
				slot.silentRejects += row.SilentRejectCount / int64(len(categories))
			}
			totalsByCampaign[row.CampaignID] += row.EventCount / int64(len(categories))
		}
	}
	out := make([]reports.CustomerFraudByTypeRowDTO, 0, len(byKey))
	for key, slot := range byKey {
		total := totalsByCampaign[key.CampaignID]
		sharePct := 0.0
		if total > 0 {
			sharePct = float64(slot.events) / float64(total) * 100
		}
		ratio := calcSilentRejectRatio(slot.silentRejects, slot.events)
		out = append(out, reports.CustomerFraudByTypeRowDTO{
			CampaignID:         key.CampaignID,
			FraudCategory:      key.Category,
			FraudCategoryLabel: FraudCategoryLabel(key.Category),
			EventCount:         slot.events,
			SilentRejectCount:  slot.silentRejects,
			SharePct:           sharePct,
			ShareLabel:         reports.FormatShareLabel(sharePct / 100),
			SilentRejectRatio:  ratio,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].EventCount == out[j].EventCount {
			return out[i].FraudCategory < out[j].FraudCategory
		}
		return out[i].EventCount > out[j].EventCount
	})
	return out
}

func QueryCustomerFraudOverview(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (totalEvents, blockedEvents, silentRejectEvents int64, err error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return 0, 0, 0, nil
	}
	const q = `
SELECT
 count() AS total_events,
 countIf(silent_reject_event = 1) AS silent_reject_events,
 countIf(silent_reject_event = 0 AND fraud_reason != '') AS blocked_events
FROM fraud_events
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?`
	if err := clickhouseQuery.QueryRow(ctx, q, campaignIDs, from, to).Scan(&totalEvents, &silentRejectEvents, &blockedEvents); err != nil {
		return 0, 0, 0, err
	}
	return totalEvents, blockedEvents, silentRejectEvents, nil
}
