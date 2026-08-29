package fraud

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const fraudEvidencePackQuery = `
SELECT
 event_type,
 campaign_id,
 coalesce(JSONExtractString(payload, 'placement_id'), '') AS placement_id,
 fraud_reason,
 fraud_score,
 silent_reject_event,
 layer_desync_count,
 created_at
FROM fraud_events
WHERE click_id = ?
 AND campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
ORDER BY created_at ASC`

func registerFraudEvidencePackReport(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/fraud-evidence-pack", limit(permAny(perms, h.WrapReport("fraud-evidence-pack", func(w http.ResponseWriter, r *http.Request) { getFraudEvidencePackReport(h, w, r) }))))
}

func getFraudEvidencePackReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	if h.DenyScopedAPIKeyReport != nil && h.DenyScopedAPIKeyReport(w, r, "fraud-evidence-pack") {
		return
	}
	customerID, ok := h.ResolveReportCustomerID(w, r)
	if !ok {
		return
	}
	clickID := strings.TrimSpace(r.URL.Query().Get("click_id"))
	if clickID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "click_id required")
		return
	}
	if len(h.FraudEvidencePackHMACSecret) == 0 {
		httpresponse.Error(w, http.StatusServiceUnavailable, "EVIDENCE_SIGNING_UNAVAILABLE", "fraud evidence signing secret not configured")
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
	campaignFilter, campaignFilterSet, err := reports.ParseOptionalCampaignFilter(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if campaignFilterSet && !h.AuthorizeReportCampaign(w, r, campaignFilter) {
		return
	}

	campaignIDs, err := reports.ListCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	campaignIDs = reports.NarrowCampaignIDs(campaignIDs, campaignFilter, campaignFilterSet)
	if len(campaignIDs) == 0 {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "no evidence for click_id")
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()

	timelineEvents, err := reports.QueryClickLogTimelineCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, clickID, from, to, 200)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	fraudRows, err := queryFraudEvidencePackFraudCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, clickID, from, to)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if len(timelineEvents) == 0 && len(fraudRows) == 0 {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "no evidence for click_id")
		return
	}

	pack := reports.FraudEvidencePackDTO{
		ClickID:     clickID,
		CustomerID:  customerID.String(),
		RangeFrom:   from.UTC().Format(time.RFC3339),
		RangeTo:     to.UTC().Format(time.RFC3339),
		Timeline:    timelineToFraudEvidenceRows(timelineEvents),
		FraudEvents: fraudRows,
	}
	if campaignFilterSet {
		pack.CampaignID = campaignFilter.String()
	} else if len(campaignIDs) == 1 {
		pack.CampaignID = campaignIDs[0].String()
	}
	pack.Signals = aggregateFraudEvidenceSignals(fraudRows)

	signed, err := BuildSignedFraudEvidencePack(h.FraudEvidencePackHMACSecret, pack)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, signed)
}

func timelineToFraudEvidenceRows(events []reports.ClickLogEventDTO) []reports.FraudEvidenceTimelineRowDTO {
	if len(events) == 0 {
		return []reports.FraudEvidenceTimelineRowDTO{}
	}
	out := make([]reports.FraudEvidenceTimelineRowDTO, 0, len(events))
	for i := range events {
		out = append(out, reports.FraudEvidenceTimelineRowDTO{
			EventType:   events[i].EventType,
			CampaignID:  events[i].CampaignID,
			PlacementID: events[i].PlacementID,
			CreatedAt:   events[i].CreatedAt,
			Country:     events[i].Country,
			Sub1:        events[i].Sub1,
		})
	}
	return out
}

func queryFraudEvidencePackFraudCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	clickID string,
	from, to time.Time,
) ([]reports.FraudEvidenceFraudRowDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 || clickID == "" {
		return nil, nil
	}
	rows, err := clickhouseQuery.Query(ctx, fraudEvidencePackQuery, clickID, campaignIDs, from, to)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]reports.FraudEvidenceFraudRowDTO, 0, 4)
	for rows.Next() {
		var row reports.FraudEvidenceFraudRowDTO
		var silent uint8
		var createdAt time.Time
		if err := rows.Scan(
			&row.EventType,
			&row.CampaignID,
			&row.PlacementID,
			&row.FraudReason,
			&row.FraudScore,
			&silent,
			&row.LayerDesyncCount,
			&createdAt,
		); err != nil {
			return nil, err
		}
		row.SilentRejectEvent = silent == 1
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		out = append(out, row)
	}
	return out, rows.Err()
}
