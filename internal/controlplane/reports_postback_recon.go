package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostbackReconRowDTO struct {
	CampaignID           string `json:"campaign_id"`
	ClickID              string `json:"click_id"`
	ConversionAt         string `json:"conversion_at"`
	ConversionValueMicro int64  `json:"conversion_value_micro,omitempty"`
	LedgerDayFeeMicro    int64  `json:"ledger_day_fee_micro,omitempty"`
	PostbackStatus       string `json:"postback_status"`
	ReconcileStatus      string `json:"reconcile_status"`
	ErrorMessage         string `json:"error_message,omitempty"`
}

type PostbackReconReportResponse struct {
	Rows       []PostbackReconRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

const postbackReconClickHouseQuery = `
SELECT
 campaign_id,
 click_id,
 max(created_at) AS conversion_at,
 max(JSONExtractInt(payload, 'revenue_micro')) AS conversion_value_micro
FROM conversions
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
GROUP BY campaign_id, click_id
ORDER BY conversion_at DESC
LIMIT ? OFFSET ?`

const postbackReconCHCountQuery = `
SELECT count() FROM (
 SELECT campaign_id, click_id
 FROM conversions
 WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
 GROUP BY campaign_id, click_id
)`

func (h *ReportsHTTPHandlers) registerPostbackReconReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/postback-reconciliation", limit(permAny(perms, h.wrapReport("postback-reconciliation", h.getPostbackReconReport))))
}

func (h *ReportsHTTPHandlers) getPostbackReconReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if h.Pool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "postgres not configured")
		return
	}
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
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
		httpresponse.JSON(w, http.StatusOK, PostbackReconReportResponse{
			Rows:      []PostbackReconRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	convRows, total, err := queryPostbackReconCHRows(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(convRows) == 0 {
		httpresponse.JSON(w, http.StatusOK, PostbackReconReportResponse{
			Rows:      []PostbackReconRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	pgCtx, pgCancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer pgCancel()
	dispatchByClick, err := queryPostbackDispatchByClickIDs(pgCtx, h.Pool, convRows)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	ledgerDayFees, err := queryCampaignDayLedgerFees(pgCtx, h.Pool, convRows, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	out := make([]PostbackReconRowDTO, 0, len(convRows))
	for _, conv := range convRows {
		dispatch := dispatchByClick[postbackDispatchKey(conv.campaignID, conv.clickID)]
		out = append(out, toPostbackReconRowDTO(conv, dispatch, ledgerDayFees[campaignDayKey(conv.campaignID, conv.at)]))
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, PostbackReconReportResponse{
		Rows:       out,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

type postbackReconCHRow struct {
	campaignID           uuid.UUID
	clickID              string
	at                   time.Time
	conversionValueMicro int64
}

type postbackDispatchRow struct {
	status       string
	errorMessage string
}

func postbackDispatchKey(campaignID uuid.UUID, clickID string) string {
	return campaignID.String() + "|" + clickID
}

func queryPostbackReconCHRows(
	ctx context.Context,
	clickhouseQuery *database.CHQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]postbackReconCHRow, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, postbackReconCHCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, postbackReconClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]postbackReconCHRow, 0, limit)
	for clickhouseRows.Next() {
		var row postbackReconCHRow
		if err := clickhouseRows.Scan(&row.campaignID, &row.clickID, &row.at, &row.conversionValueMicro); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}

func queryPostbackDispatchByClickIDs(
	ctx context.Context,
	pool *pgxpool.Pool,
	convRows []postbackReconCHRow,
) (map[string]postbackDispatchRow, error) {
	if pool == nil || len(convRows) == 0 {
		return map[string]postbackDispatchRow{}, nil
	}
	clickIDs := make([]string, 0, len(convRows))
	campaignIDs := make([]uuid.UUID, 0, len(convRows))
	for _, row := range convRows {
		clickIDs = append(clickIDs, row.clickID)
		campaignIDs = append(campaignIDs, row.campaignID)
	}
	rows, err := pool.Query(ctx, `
SELECT campaign_id, click_id, status, COALESCE(error_message, '')
FROM postback_dispatches
WHERE click_id = ANY($1::text[])
 AND campaign_id = ANY($2::uuid[])`, clickIDs, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]postbackDispatchRow, len(convRows))
	for rows.Next() {
		var campaignID uuid.UUID
		var clickID, status, errMsg string
		if err := rows.Scan(&campaignID, &clickID, &status, &errMsg); err != nil {
			return nil, err
		}
		key := postbackDispatchKey(campaignID, clickID)
		out[key] = postbackDispatchRow{status: status, errorMessage: errMsg}
	}
	return out, rows.Err()
}

func toPostbackReconRowDTO(conv postbackReconCHRow, dispatch postbackDispatchRow, ledgerDayFeeMicro int64) PostbackReconRowDTO {
	status := dispatch.status
	reconcile := postbackReconcileStatus(status)
	return PostbackReconRowDTO{
		CampaignID:           conv.campaignID.String(),
		ClickID:              conv.clickID,
		ConversionAt:         conv.at.UTC().Format(time.RFC3339),
		ConversionValueMicro: conv.conversionValueMicro,
		LedgerDayFeeMicro:    ledgerDayFeeMicro,
		PostbackStatus:       status,
		ReconcileStatus:      reconcile,
		ErrorMessage:         dispatch.errorMessage,
	}
}

func postbackReconcileStatus(postbackStatus string) string {
	switch postbackStatus {
	case "SENT":
		return "ok"
	case "FAILED", "IN_FLIGHT":
		return postbackStatus
	case "":
		return "missing_postback"
	default:
		return "unknown"
	}
}

func campaignDayKey(campaignID uuid.UUID, at time.Time) string {
	return campaignID.String() + "|" + at.UTC().Format("2006-01-02")
}

func queryCampaignDayLedgerFees(
	ctx context.Context,
	pool *pgxpool.Pool,
	convRows []postbackReconCHRow,
	from, to time.Time,
) (map[string]int64, error) {
	if pool == nil || len(convRows) == 0 {
		return map[string]int64{}, nil
	}
	campaignIDs := make([]uuid.UUID, 0, len(convRows))
	seen := make(map[uuid.UUID]struct{}, len(convRows))
	for _, row := range convRows {
		if _, ok := seen[row.campaignID]; ok {
			continue
		}
		seen[row.campaignID] = struct{}{}
		campaignIDs = append(campaignIDs, row.campaignID)
	}
	rows, err := pool.Query(ctx, `
SELECT campaign_id, created_at::date AS day, COALESCE(SUM(ABS(amount)), 0)::bigint AS fee_micro
FROM balance_ledger
WHERE campaign_id = ANY($1::uuid[])
 AND type = 'FEE'
 AND created_at >= $2
 AND created_at < $3
GROUP BY campaign_id, day`, campaignIDs, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var campaignID uuid.UUID
		var day time.Time
		var feeMicro int64
		if err := rows.Scan(&campaignID, &day, &feeMicro); err != nil {
			return nil, err
		}
		out[campaignDayKey(campaignID, day)] = feeMicro
	}
	return out, rows.Err()
}

func queryPostbackReconExportRows(
	ctx context.Context,
	deps ReportExportDeps,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]PostbackReconRowDTO, int64, error) {
	if deps.Pool == nil || deps.ClickHouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	convRows, total, err := queryPostbackReconCHRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(convRows) == 0 {
		return nil, total, nil
	}
	dispatchByClick, err := queryPostbackDispatchByClickIDs(ctx, deps.Pool, convRows)
	if err != nil {
		return nil, 0, err
	}
	ledgerDayFees, err := queryCampaignDayLedgerFees(ctx, deps.Pool, convRows, from, to)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PostbackReconRowDTO, 0, len(convRows))
	for _, conv := range convRows {
		dispatch := dispatchByClick[postbackDispatchKey(conv.campaignID, conv.clickID)]
		out = append(out, toPostbackReconRowDTO(conv, dispatch, ledgerDayFees[campaignDayKey(conv.campaignID, conv.at)]))
	}
	return out, total, nil
}
