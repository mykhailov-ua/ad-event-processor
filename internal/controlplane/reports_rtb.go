package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type RtbOverviewRowDTO struct {
	DealID     string  `json:"deal_id,omitempty"`
	Bids       int64   `json:"bids"`
	Wins       int64   `json:"wins"`
	WinRate    float64 `json:"win_rate"`
	SpendMicro int64   `json:"spend_micro"`
}

type RtbNoBidReasonRowDTO struct {
	NoBidReason string `json:"no_bid_reason"`
	BidCount    int64  `json:"bid_count"`
}

type RtbOverviewReportResponse struct {
	Rows       []RtbOverviewRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO    `json:"freshness"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type RtbNoBidReasonsReportResponse struct {
	Rows       []RtbNoBidReasonRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type RtbGeoDeviceRowDTO struct {
	Country    string  `json:"country"`
	DeviceOS   string  `json:"device_os"`
	Bids       int64   `json:"bids"`
	Wins       int64   `json:"wins"`
	WinRate    float64 `json:"win_rate"`
	SpendMicro int64   `json:"spend_micro"`
}

type RtbGeoDeviceReportResponse struct {
	Rows       []RtbGeoDeviceRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO     `json:"freshness"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

const rtbOverviewQuery = `
SELECT
 deal_id,
 count() AS bids,
 sum(won) AS wins,
 sumIf(price_micro, won = 1) AS spend_micro
FROM rtb_exchange_log
WHERE created_at >= ?
 AND created_at < ?
GROUP BY deal_id
ORDER BY bids DESC
LIMIT ? OFFSET ?`

const rtbOverviewCountQuery = `
SELECT count() FROM (
 SELECT deal_id
 FROM rtb_exchange_log
 WHERE created_at >= ?
 AND created_at < ?
 GROUP BY deal_id
)`

const rtbNoBidReasonsQuery = `
SELECT
 no_bid_reason,
 count() AS bid_count
FROM rtb_exchange_log
WHERE created_at >= ?
 AND created_at < ?
 AND won = 0
 AND no_bid_reason > 0
GROUP BY no_bid_reason
ORDER BY bid_count DESC
LIMIT ? OFFSET ?`

const rtbNoBidReasonsCountQuery = `
SELECT count() FROM (
 SELECT no_bid_reason
 FROM rtb_exchange_log
 WHERE created_at >= ?
 AND created_at < ?
 AND won = 0
 AND no_bid_reason > 0
 GROUP BY no_bid_reason
)`

const rtbGeoDeviceQuery = `
SELECT
 geo_country,
 device_os,
 count() AS bids,
 sum(won) AS wins,
 sumIf(price_micro, won = 1) AS spend_micro
FROM rtb_exchange_log
WHERE created_at >= ?
 AND created_at < ?
GROUP BY geo_country, device_os
ORDER BY bids DESC
LIMIT ? OFFSET ?`

const rtbGeoDeviceCountQuery = `
SELECT count() FROM (
 SELECT geo_country, device_os
 FROM rtb_exchange_log
 WHERE created_at >= ?
 AND created_at < ?
 GROUP BY geo_country, device_os
)`

func (reports *ReportsHTTPHandlers) registerRtbReports(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/rtb/overview", limit(perm("rtb:read", reports.wrapReport("rtb-overview", reports.getRtbOverviewReport))))
	mux.HandleFunc("GET /api/v1/reports/rtb/no-bid-reasons", limit(perm("rtb:read", reports.wrapReport("rtb-no-bid-reasons", reports.getRtbNoBidReasonsReport))))
	mux.HandleFunc("GET /api/v1/reports/rtb/geo-device", limit(perm("rtb:read", reports.wrapReport("rtb-geo-device", reports.getRtbGeoDeviceReport))))
}

func (reports *ReportsHTTPHandlers) getRtbOverviewReport(w http.ResponseWriter, r *http.Request) {
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 500)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryRtbOverviewRows(chCtx, reports.CHQuery, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, RtbOverviewReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func (reports *ReportsHTTPHandlers) getRtbNoBidReasonsReport(w http.ResponseWriter, r *http.Request) {
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 500)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryRtbNoBidReasonRows(chCtx, reports.CHQuery, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, RtbNoBidReasonsReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func (reports *ReportsHTTPHandlers) getRtbGeoDeviceReport(w http.ResponseWriter, r *http.Request) {
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 500)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryRtbGeoDeviceRows(chCtx, reports.CHQuery, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, RtbGeoDeviceReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryRtbOverviewRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	from, to time.Time,
	limit, offset int,
) ([]RtbOverviewRowDTO, int64, error) {
	if chQuery == nil {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, rtbOverviewCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, rtbOverviewQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	out := make([]RtbOverviewRowDTO, 0, limit)
	for chRows.Next() {
		var row RtbOverviewRowDTO
		if err := chRows.Scan(&row.DealID, &row.Bids, &row.Wins, &row.SpendMicro); err != nil {
			return nil, 0, err
		}
		if row.Bids > 0 {
			row.WinRate = calcRtbWinRate(row.Wins, row.Bids)
		}
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func queryRtbGeoDeviceRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	from, to time.Time,
	limit, offset int,
) ([]RtbGeoDeviceRowDTO, int64, error) {
	if chQuery == nil {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, rtbGeoDeviceCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, rtbGeoDeviceQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	out := make([]RtbGeoDeviceRowDTO, 0, limit)
	for chRows.Next() {
		var row RtbGeoDeviceRowDTO
		if err := chRows.Scan(&row.Country, &row.DeviceOS, &row.Bids, &row.Wins, &row.SpendMicro); err != nil {
			return nil, 0, err
		}
		if row.Bids > 0 {
			row.WinRate = calcRtbWinRate(row.Wins, row.Bids)
		}
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func queryRtbNoBidReasonRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	from, to time.Time,
	limit, offset int,
) ([]RtbNoBidReasonRowDTO, int64, error) {
	if chQuery == nil {
		return nil, 0, nil
	}
	var total int64
	if err := chQuery.QueryRow(ctx, rtbNoBidReasonsCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, rtbNoBidReasonsQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	out := make([]RtbNoBidReasonRowDTO, 0, limit)
	for chRows.Next() {
		var reasonCode uint16
		var row RtbNoBidReasonRowDTO
		if err := chRows.Scan(&reasonCode, &row.BidCount); err != nil {
			return nil, 0, err
		}
		row.NoBidReason = rtb.NoBidReason(reasonCode).String()
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func calcRtbWinRate(wins, bids int64) float64 {
	if bids <= 0 {
		return 0
	}
	return float64(wins) / float64(bids)
}
