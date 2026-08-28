package reports

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClickLogEventDTO struct {
	EventType           string `json:"event_type"`
	ClickID             string `json:"click_id"`
	CampaignID          string `json:"campaign_id"`
	PlacementID         string `json:"placement_id,omitempty"`
	CreatedAt           string `json:"created_at"`
	AttributedCostMicro int64  `json:"attributed_cost_micro,omitempty"`
	CostSource          string `json:"cost_source,omitempty"`
	RevenueMicro        int64  `json:"revenue_micro,omitempty"`
	InboundStatus       string `json:"inbound_status,omitempty"`
	GoalName            string `json:"goal_name,omitempty"`
	Sub1                string `json:"sub1,omitempty"`
	Country             string `json:"country,omitempty"`
}

type ClickLogPostbackDTO struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type ClickLogReportResponse struct {
	Events     []ClickLogEventDTO    `json:"events"`
	Postbacks  []ClickLogPostbackDTO `json:"postbacks,omitempty"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

const clickLogBrowseCountQuery = `
SELECT count()
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?`

const clickLogBrowseQuery = `
SELECT
 click_id,
 campaign_id,
 placement_id,
 created_at,
 attributed_cost_micro,
 cost_source,
 payload
FROM clicks
WHERE campaign_id IN (?)
 AND created_at >= ?
 AND created_at < ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?`

const clickLogTimelineQuery = `
SELECT * FROM (
 SELECT
  'impression' AS event_type,
  click_id,
  campaign_id,
  placement_id,
  created_at,
  toInt64(0) AS attributed_cost_micro,
  '' AS cost_source,
  payload
 FROM impressions
 WHERE click_id = ?
  AND campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
 UNION ALL
 SELECT
  'click' AS event_type,
  click_id,
  campaign_id,
  placement_id,
  created_at,
  attributed_cost_micro,
  cost_source,
  payload
 FROM clicks
 WHERE click_id = ?
  AND campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
 UNION ALL
 SELECT
  'conversion' AS event_type,
  click_id,
  campaign_id,
  placement_id,
  created_at,
  toInt64(0) AS attributed_cost_micro,
  '' AS cost_source,
  payload
 FROM conversions
 WHERE click_id = ?
  AND campaign_id IN (?)
  AND created_at >= ?
  AND created_at < ?
)
ORDER BY created_at ASC
LIMIT ?`

func (h *ReportsHTTPHandlers) registerClickLogReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "campaigns:read"}
	mux.HandleFunc("GET /api/v1/reports/click-log", limit(permAny(perms, h.wrapReport("click-log", h.getClickLogReport))))
	mux.HandleFunc("GET /api/v1/reports/clicks", limit(permAny(perms, h.wrapReport("click-log", h.getClickLogReport))))
}

func (h *ReportsHTTPHandlers) getClickLogReport(w http.ResponseWriter, r *http.Request) {
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
	clickID := strings.TrimSpace(r.URL.Query().Get("click_id"))
	campaignFilter, campaignFilterSet, err := parseOptionalCampaignFilter(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	if clickID == "" && !campaignFilterSet {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "click_id or campaign_id required")
		return
	}
	if campaignFilterSet && !h.authorizeReportCampaign(w, r, campaignFilter) {
		return
	}

	campaignIDs, err := listCustomerCampaignIDs(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	campaignIDs = narrowCampaignIDs(campaignIDs, campaignFilter, campaignFilterSet)
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ClickLogReportResponse{
			Events:    []ClickLogEventDTO{},
			Postbacks: []ClickLogPostbackDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	page, err := coldpath.ParseCursorPagination(r, 50, 200)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	if clickID != "" {
		page.Limit = 200
		page.Offset = 0
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()

	var events []ClickLogEventDTO
	var total int64
	if clickID != "" {
		events, err = queryClickLogTimelineCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, clickID, from, to, page.Limit)
		total = int64(len(events))
	} else {
		events, total, err = queryClickLogBrowseCH(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to, page.Limit, page.Offset)
	}
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var postbacks []ClickLogPostbackDTO
	if clickID != "" && h.Pool != nil {
		pgCtx, pgCancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
		postbacks, err = queryClickLogPostbacks(pgCtx, h.Pool, campaignIDs, clickID)
		pgCancel()
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	if postbacks == nil {
		postbacks = []ClickLogPostbackDTO{}
	}
	if events == nil {
		events = []ClickLogEventDTO{}
	}

	var nextCursor string
	if clickID == "" && int64(page.Offset)+int64(len(events)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, ClickLogReportResponse{
		Events:     events,
		Postbacks:  postbacks,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func parseOptionalCampaignFilter(r *http.Request) (uuid.UUID, bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("campaign_id"))
	if raw == "" {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}

func narrowCampaignIDs(all []uuid.UUID, filter uuid.UUID, filterSet bool) []uuid.UUID {
	if !filterSet {
		return all
	}
	for _, id := range all {
		if id == filter {
			return []uuid.UUID{filter}
		}
	}
	return nil
}

type clickLogEventRow struct {
	eventType           string
	clickID             string
	campaignID          uuid.UUID
	placementID         string
	at                  time.Time
	attributedCostMicro int64
	costSource          string
	payload             string
}

func queryClickLogBrowseCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]ClickLogEventDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, clickLogBrowseCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := clickhouseQuery.Query(ctx, clickLogBrowseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ClickLogEventDTO, 0, limit)
	for rows.Next() {
		var row clickLogEventRow
		if err := rows.Scan(&row.clickID, &row.campaignID, &row.placementID, &row.at, &row.attributedCostMicro, &row.costSource, &row.payload); err != nil {
			return nil, 0, err
		}
		row.eventType = "click"
		out = append(out, toClickLogEventDTO(row))
	}
	return out, total, rows.Err()
}

func queryClickLogTimelineCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	clickID string,
	from, to time.Time,
	limit int,
) ([]ClickLogEventDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 || clickID == "" {
		return nil, nil
	}
	rows, err := clickhouseQuery.Query(
		ctx,
		clickLogTimelineQuery,
		clickID, campaignIDs, from, to,
		clickID, campaignIDs, from, to,
		clickID, campaignIDs, from, to,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]ClickLogEventDTO, 0, 8)
	for rows.Next() {
		var row clickLogEventRow
		if err := rows.Scan(
			&row.eventType,
			&row.clickID,
			&row.campaignID,
			&row.placementID,
			&row.at,
			&row.attributedCostMicro,
			&row.costSource,
			&row.payload,
		); err != nil {
			return nil, err
		}
		out = append(out, toClickLogEventDTO(row))
	}
	return out, rows.Err()
}

func queryClickLogPostbacks(ctx context.Context, pool *pgxpool.Pool, campaignIDs []uuid.UUID, clickID string) ([]ClickLogPostbackDTO, error) {
	if pool == nil || clickID == "" || len(campaignIDs) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT status, COALESCE(error_message, ''), created_at
FROM postback_dispatches
WHERE click_id = $1
 AND campaign_id = ANY($2::uuid[])
ORDER BY created_at ASC`, clickID, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ClickLogPostbackDTO, 0, 4)
	for rows.Next() {
		var status, errMsg string
		var at time.Time
		if err := rows.Scan(&status, &errMsg, &at); err != nil {
			return nil, err
		}
		out = append(out, ClickLogPostbackDTO{
			Status:       status,
			ErrorMessage: errMsg,
			CreatedAt:    at.UTC().Format(time.RFC3339),
		})
	}
	return out, rows.Err()
}

func toClickLogEventDTO(row clickLogEventRow) ClickLogEventDTO {
	sub1, country, status, goalName, revenueMicro := clickLogPayloadFields(row.payload)
	dto := ClickLogEventDTO{
		EventType:           row.eventType,
		ClickID:             row.clickID,
		CampaignID:          row.campaignID.String(),
		PlacementID:         row.placementID,
		CreatedAt:           row.at.UTC().Format(time.RFC3339),
		AttributedCostMicro: row.attributedCostMicro,
		CostSource:          row.costSource,
		Sub1:                sub1,
		Country:             country,
		InboundStatus:       status,
		GoalName:            goalName,
		RevenueMicro:        revenueMicro,
	}
	return dto
}

func clickLogPayloadFields(payload string) (sub1, country, status, goalName string, revenueMicro int64) {
	if payload == "" {
		return "", "", "", "", 0
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return "", "", "", "", 0
	}
	readString := func(key string) string {
		val, ok := raw[key]
		if !ok || len(val) == 0 || string(val) == "null" {
			return ""
		}
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			return s
		}
		return strings.TrimSpace(string(val))
	}
	sub1 = readString("sub1")
	country = readString("country")
	for _, key := range []string{"status", "affiliate_status", "conversion_status"} {
		if v := readString(key); v != "" {
			status = v
			break
		}
	}
	goalName = readString("goal_name")
	if val, ok := raw["revenue_micro"]; ok {
		_ = json.Unmarshal(val, &revenueMicro)
	}
	return sub1, country, status, goalName, revenueMicro
}
