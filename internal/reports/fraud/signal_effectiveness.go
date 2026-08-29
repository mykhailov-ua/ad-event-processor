package fraud

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func registerSignalEffectivenessReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read", permCampaignsReadMasked}
	mux.HandleFunc("GET /api/v1/reports/signal-effectiveness", limit(permAny(perms, h.WrapReport("signal-effectiveness", func(w http.ResponseWriter, r *http.Request) { getSignalEffectivenessReport(h, w, r) }))))
}

func getSignalEffectivenessReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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
	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, reports.SignalEffectivenessReportResponse{
			Rows:      []reports.SignalEffectivenessRowDTO{},
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
	rows := aggregateSignalEffectiveness(rawRows, r.Context())
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
	httpresponse.JSON(w, http.StatusOK, reports.SignalEffectivenessReportResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func aggregateSignalEffectiveness(rows []reports.FraudBreakdownRowDTO, scrubCtx context.Context) []reports.SignalEffectivenessRowDTO {
	type slot struct {
		events  int64
		silent  int64
		blocked int64
	}
	bySignal := make(map[string]*slot)
	for _, row := range rows {
		if !isWireSignalReason(row.FraudReason) {
			continue
		}
		for _, code := range strings.Split(row.FraudReason, ",") {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			s := bySignal[code]
			if s == nil {
				s = &slot{}
				bySignal[code] = s
			}
			s.events += row.EventCount
			s.silent += row.SilentRejectCount
			s.blocked += row.EventCount - row.SilentRejectCount
		}
	}
	out := make([]reports.SignalEffectivenessRowDTO, 0, len(bySignal))
	for code, s := range bySignal {
		blockRate := 0.0
		silentRate := 0.0
		if s.events > 0 {
			blockRate = float64(s.blocked) / float64(s.events)
			silentRate = float64(s.silent) / float64(s.events)
		}
		category, label := FraudReasonToCategory(code)
		row := reports.SignalEffectivenessRowDTO{
			EventVolume:         s.events,
			BlockRate:           blockRate,
			BlockRateDisplay:    reports.FormatRateDisplay(blockRate),
			SilentRejectRate:    silentRate,
			SilentRejectDisplay: reports.FormatRateDisplay(silentRate),
			SuggestedWeightTier: suggestedWeightTier(s.events, blockRate),
		}
		if maskLevelFromContext(scrubCtx) == authz.MaskFull {
			row.SignalCode = code
		} else {
			row.FraudCategory = category
			row.FraudCategoryLabel = label
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EventVolume > out[j].EventVolume
	})
	return out
}

func suggestedWeightTier(volume int64, blockRate float64) string {
	switch {
	case volume >= 1000 && blockRate >= 0.1:
		return "high"
	case volume >= 100 && blockRate >= 0.03:
		return "medium"
	default:
		return "low"
	}
}
