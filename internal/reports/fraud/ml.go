package fraud

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

const (
	mlScoreDistributionQuery = `
SELECT floor(score * 10) / 10 AS score_bucket, count() AS row_count
FROM ml_shadow_scores
WHERE created_at >= ? AND created_at < ?
GROUP BY score_bucket
ORDER BY score_bucket
LIMIT ? OFFSET ?`

	mlScoreDistributionCountQuery = `
SELECT count(DISTINCT floor(score * 10) / 10)
FROM ml_shadow_scores
WHERE created_at >= ? AND created_at < ?`

	mlShadowDeltaQuery = `
SELECT
 toStartOfHour(s.created_at) AS bucket,
 avg(s.score) AS avg_shadow_score,
 count() AS score_count,
 avg(f.events) AS avg_feature_events
FROM ml_shadow_scores AS s
LEFT JOIN (
 SELECT window_start, ip_hash, sum(events) AS events
 FROM ml_features_1m
 WHERE window_start >= ? AND window_start < ?
 GROUP BY window_start, ip_hash
) AS f ON f.window_start = toStartOfMinute(s.created_at) AND f.ip_hash = s.ip_hash
WHERE s.created_at >= ? AND s.created_at < ?
GROUP BY bucket
ORDER BY bucket
LIMIT ? OFFSET ?`

	mlShadowDeltaCountQuery = `
SELECT count(DISTINCT toStartOfHour(created_at))
FROM ml_shadow_scores
WHERE created_at >= ? AND created_at < ?`

	mlFeatureSpikesQuery = `
SELECT
 window_start,
 sum(events) AS events,
 sum(clicks) AS clicks,
 uniqExact(campaign_id) AS campaigns
FROM ml_features_1m
WHERE window_start >= ? AND window_start < ?
GROUP BY window_start
ORDER BY events DESC
LIMIT ? OFFSET ?`

	mlFeatureSpikesCountQuery = `
SELECT count(DISTINCT window_start)
FROM ml_features_1m
WHERE window_start >= ? AND window_start < ?`
)

func registerMLReports(h *reports.ReportsHTTPHandlers, mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/reports/ml/score-distribution", limit(perm("shards:read", h.WrapReport("ml/score-distribution", func(w http.ResponseWriter, r *http.Request) { getMLScoreDistributionReport(h, w, r) }))))
	mux.HandleFunc("GET /api/v1/reports/ml/shadow-delta", limit(perm("shards:read", h.WrapReport("ml/shadow-delta", func(w http.ResponseWriter, r *http.Request) { getMLShadowDeltaReport(h, w, r) }))))
	mux.HandleFunc("GET /api/v1/reports/ml/feature-spikes", limit(perm("shards:read", h.WrapReport("ml/feature-spikes", func(w http.ResponseWriter, r *http.Request) { getMLFeatureSpikesReport(h, w, r) }))))
}

func getMLScoreDistributionReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	writeMLReport(h, w, r, queryMLScoreDistributionRows)
}

func getMLShadowDeltaReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeMLReport(h, w, r, QueryMLShadowDeltaRows)
		return
	}
	snap, ok, err := LoadMLShadowDeltaSnapshot(r.Context(), h.Pool)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	if !ok {
		writeMLReport(h, w, r, QueryMLShadowDeltaRows)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	rows, total := PaginateMLShadowDeltaSnapshotRows(snap.Rows, page.Limit, page.Offset)
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.ReportRowsResponse{
		Rows:       rows,
		Freshness:  MLShadowDeltaSnapshotFreshness(snap, time.Now().UTC()),
		NextCursor: nextCursor,
	})
}

func getMLFeatureSpikesReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	writeMLReport(h, w, r, queryMLFeatureSpikeRows)
}

type mlReportQueryFunc func(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]map[string]any, int64, error)

func writeMLReport(h *reports.ReportsHTTPHandlers, w http.ResponseWriter, r *http.Request, queryFn mlReportQueryFunc) {
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
	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, total, err := queryFn(clickhouseCtx, h.ClickHouseQuery, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.WriteReportsHandlerError(h, w, err)
		return
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, reports.ReportRowsResponse{
		Rows:       rows,
		Freshness:  h.ReportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryMLScoreDistributionRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, mlScoreDistributionCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := clickhouseQuery.Query(ctx, mlScoreDistributionQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var bucket float64
		var count int64
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"score_bucket": bucket,
			"row_count":    count,
		})
	}
	return out, total, rows.Err()
}

func QueryMLShadowDeltaRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, mlShadowDeltaCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := clickhouseQuery.Query(ctx, mlShadowDeltaQuery, from, to, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var bucket time.Time
		var avgScore, avgEvents float64
		var count int64
		if err := rows.Scan(&bucket, &avgScore, &count, &avgEvents); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"bucket":             bucket.UTC().Format(time.RFC3339),
			"avg_shadow_score":   avgScore,
			"score_count":        count,
			"avg_feature_events": avgEvents,
		})
	}
	return out, total, rows.Err()
}

func queryMLFeatureSpikeRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	from, to time.Time,
	limit, offset int,
) ([]map[string]any, int64, error) {
	var total int64
	if err := clickhouseQuery.QueryRow(ctx, mlFeatureSpikesCountQuery, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := clickhouseQuery.Query(ctx, mlFeatureSpikesQuery, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var windowStart time.Time
		var events, clicks, campaigns int64
		if err := rows.Scan(&windowStart, &events, &clicks, &campaigns); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"window_start": windowStart.UTC().Format(time.RFC3339),
			"events":       events,
			"clicks":       clicks,
			"campaigns":    campaigns,
		})
	}
	return out, total, rows.Err()
}

func mlReportKeys() []string {
	return []string{"ml/score-distribution", "ml/shadow-delta", "ml/feature-spikes"}
}

func validateMLReportRoute(key string) error {
	for _, allowed := range mlReportKeys() {
		if key == allowed {
			return nil
		}
	}
	return fmt.Errorf("unknown ml report %q", key)
}
