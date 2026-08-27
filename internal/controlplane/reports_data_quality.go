package controlplane

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DataQualityRowDTO struct {
	CampaignID string  `json:"campaign_id"`
	Date       string  `json:"date"`
	PGTotal    int64   `json:"pg_total"`
	CHTotal    int64   `json:"ch_total"`
	DiffPct    float64 `json:"diff_pct"`
	Severity   string  `json:"severity"`
}

type DataQualityReportResponse struct {
	Rows       []DataQualityRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO    `json:"freshness"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

func dataQualitySeverity(pgTotal int64, diffPct float64) string {
	if pgTotal == 0 {
		return "info"
	}
	switch {
	case diffPct > 0.05:
		return "high"
	case diffPct > hyg30CHStatsTolerancePct:
		return "medium"
	default:
		return "ok"
	}
}

func (h *ReportsHTTPHandlers) registerDataQualityReport(mux *http.ServeMux) {
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"audit:read", "customers:read"}
	mux.HandleFunc("GET /api/v1/reports/data-quality", limit(permAny(perms, h.wrapReport("data-quality", h.getDataQualityReport))))
}

func (h *ReportsHTTPHandlers) getDataQualityReport(w http.ResponseWriter, r *http.Request) {
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
		httpresponse.JSON(w, http.StatusOK, DataQualityReportResponse{
			Rows:      []DataQualityRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	pgRows, err := queryPGCampaignDailyTotals(r.Context(), h.Pool, customerID, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(pgRows) == 0 {
		httpresponse.JSON(w, http.StatusOK, DataQualityReportResponse{
			Rows:      []DataQualityRowDTO{},
			Freshness: h.reportFreshness(r.Context()),
		})
		return
	}

	clickhouseCtx, cancel := context.WithTimeout(r.Context(), reportClickHouseQueryTimeout)
	defer cancel()
	clickhouseTotals, err := queryClickHouseCampaignDailyEventTotals(clickhouseCtx, h.ClickHouseQuery, campaignIDs, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	out := buildDataQualityRows(pgRows, clickhouseTotals)

	total := int64(len(out))
	if page.Offset >= len(out) {
		out = nil
	} else {
		end := page.Offset + page.Limit
		if end > len(out) {
			end = len(out)
		}
		out = out[page.Offset:end]
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, DataQualityReportResponse{
		Rows:       out,
		Freshness:  h.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

type pgCampaignDailyTotal struct {
	campaignID uuid.UUID
	day        time.Time
	pgTotal    int64
}

func queryPGCampaignDailyTotals(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	from, to time.Time,
) ([]pgCampaignDailyTotal, error) {
	rows, err := pool.Query(ctx, `
		SELECT cs.campaign_id, cs.date,
		 cs.impressions_count + cs.clicks_count + cs.conversions_count AS pg_total
		FROM campaign_stats cs
		INNER JOIN campaigns c ON c.id = cs.campaign_id
		WHERE c.customer_id = $1
		 AND c.deleted_at IS NULL
		 AND cs.date >= $2::date
		 AND cs.date < $3::date
		 AND (cs.impressions_count + cs.clicks_count + cs.conversions_count) > 0
		ORDER BY cs.date DESC, cs.campaign_id`,
		customerID,
		pgtype.Date{Time: from.UTC(), Valid: true},
		pgtype.Date{Time: to.UTC(), Valid: true},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []pgCampaignDailyTotal
	for rows.Next() {
		var row pgCampaignDailyTotal
		if err := rows.Scan(&row.campaignID, &row.day, &row.pgTotal); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func buildDataQualityRows(
	pgRows []pgCampaignDailyTotal,
	clickhouseTotals map[string]uint64,
) []DataQualityRowDTO {
	out := make([]DataQualityRowDTO, 0, len(pgRows))
	for _, row := range pgRows {
		chTotal := int64(clickhouseTotals[campaignDailyTotalKey(row.campaignID, row.day)])
		var diffPct float64
		if row.pgTotal > 0 {
			diffPct = math.Abs(float64(chTotal-row.pgTotal)) / float64(row.pgTotal)
		}
		out = append(out, DataQualityRowDTO{
			CampaignID: row.campaignID.String(),
			Date:       row.day.UTC().Format("2006-01-02"),
			PGTotal:    row.pgTotal,
			CHTotal:    chTotal,
			DiffPct:    diffPct,
			Severity:   dataQualitySeverity(row.pgTotal, diffPct),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DiffPct == out[j].DiffPct {
			return out[i].Date > out[j].Date
		}
		return out[i].DiffPct > out[j].DiffPct
	})
	return out
}

func queryDataQualityExportRows(
	ctx context.Context,
	deps ReportExportDeps,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]DataQualityRowDTO, int64, error) {
	pgRows, err := queryPGCampaignDailyTotals(ctx, deps.Pool, customerID, from, to)
	if err != nil {
		return nil, 0, err
	}
	if len(pgRows) == 0 {
		return nil, 0, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
	defer cancel()
	clickhouseTotals, err := queryClickHouseCampaignDailyEventTotals(clickhouseCtx, deps.ClickHouseQuery, campaignIDs, from, to)
	if err != nil {
		return nil, 0, err
	}
	all := buildDataQualityRows(pgRows, clickhouseTotals)
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}
