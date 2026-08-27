package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type FilterRejectRowDTO struct {
	RejectKind  string `json:"reject_kind"`
	RejectCount int64  `json:"reject_count"`
	Country     string `json:"country,omitempty"`
	PlacementID string `json:"placement_id,omitempty"`
}

type FilterRejectReportResponse struct {
	Rows       []FilterRejectRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO     `json:"freshness"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

const filterRejectSliceQuery = `
SELECT
 reject_kind,
 country,
 placement_id,
 sum(reject_count) AS reject_count
FROM filter_reject_slices
WHERE rollup_hour >= ?
 AND rollup_hour < ?
GROUP BY reject_kind, country, placement_id
ORDER BY reject_count DESC
LIMIT ? OFFSET ?`

const filterRejectSliceCountQuery = `
SELECT count() FROM (
 SELECT reject_kind, country, placement_id
 FROM filter_reject_slices
 WHERE rollup_hour >= ?
 AND rollup_hour < ?
 GROUP BY reject_kind, country, placement_id
)`

const filterRejectQuery = `
SELECT
 reject_kind,
 sum(reject_count) AS reject_count
FROM filter_reject_rollups
WHERE rollup_hour >= ?
 AND rollup_hour < ?
GROUP BY reject_kind
ORDER BY reject_count DESC
LIMIT ? OFFSET ?`

const filterRejectCountQuery = `
SELECT count() FROM (
 SELECT reject_kind
 FROM filter_reject_rollups
 WHERE rollup_hour >= ?
 AND rollup_hour < ?
 GROUP BY reject_kind
)`

func (h *ReportsHTTPHandlers) registerFilterRejectsReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/filter-rejects", limit(perm("audit:read", h.wrapReport("filter-rejects", h.getFilterRejectsReport))))
}

func (h *ReportsHTTPHandlers) getFilterRejectsReport(w http.ResponseWriter, r *http.Request) {
	if h.ClickHouseQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 500)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	sliceMode := r.URL.Query().Get("slice") == "1" || r.URL.Query().Get("group_by") == "placement_geo"
	var rows []FilterRejectRowDTO
	var total int64
	var errQuery error
	if sliceMode {
		rows, total, errQuery = queryFilterRejectSliceRows(clickhouseCtx, h.ClickHouseQuery, from, to, page.Limit, page.Offset)
	} else {
		rows, total, errQuery = queryFilterRejectRows(clickhouseCtx, h.ClickHouseQuery, from, to, page.Limit, page.Offset)
	}
	if errQuery != nil {
		h.writeServiceError(w, errQuery)
		return
	}

	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, FilterRejectReportResponse{
		Rows:       rows,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryFilterRejectRows(
	ctx context.Context,
	clickhouseQuery *database.CHQuery,
	from, to time.Time,
	limit, offset int,
) ([]FilterRejectRowDTO, int64, error) {
	if clickhouseQuery == nil {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, filterRejectCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, filterRejectQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]FilterRejectRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row FilterRejectRowDTO
		if err := clickhouseRows.Scan(&row.RejectKind, &row.RejectCount); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}

func queryFilterRejectSliceRows(
	ctx context.Context,
	clickhouseQuery *database.CHQuery,
	from, to time.Time,
	limit, offset int,
) ([]FilterRejectRowDTO, int64, error) {
	if clickhouseQuery == nil {
		return nil, 0, nil
	}
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, filterRejectSliceCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	clickhouseRows, err := clickhouseQuery.Query(ctx, filterRejectSliceQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = clickhouseRows.Close() }()

	out := make([]FilterRejectRowDTO, 0, limit)
	for clickhouseRows.Next() {
		var row FilterRejectRowDTO
		if err := clickhouseRows.Scan(&row.RejectKind, &row.Country, &row.PlacementID, &row.RejectCount); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	return out, total, clickhouseRows.Err()
}
