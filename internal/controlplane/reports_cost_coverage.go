package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CostCoverageRowDTO struct {
	CampaignID     string `json:"campaign_id"`
	Clicks         int64  `json:"clicks"`
	SpendMicro     int64  `json:"spend_micro"`
	CoverageGap    string `json:"coverage_gap"`
	Network        string `json:"network,omitempty"`
	LastSyncStatus string `json:"last_sync_status,omitempty"`
}

const costCoverageCountQuery = `
SELECT count()
FROM (
 SELECT campaign_id
 FROM placement_stats_hourly
 WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
 GROUP BY campaign_id
 HAVING sum(clicks) > 0 AND sum(spend_micro) = 0
)`

const costCoverageQuery = `
SELECT
 campaign_id,
 sum(clicks) AS clicks,
 sum(spend_micro) AS spend_micro
FROM placement_stats_hourly
WHERE campaign_id IN (?)
 AND hour >= ?
 AND hour < ?
GROUP BY campaign_id
HAVING clicks > 0 AND spend_micro = 0
ORDER BY clicks DESC
LIMIT ? OFFSET ?`

func (reports *ReportsHTTPHandlers) registerCostCoverageReport(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perms := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/cost-sync-coverage", limit(permAny(perms, reports.wrapReport("cost-sync-coverage", reports.getCostSyncCoverageReport))))
}

func (reports *ReportsHTTPHandlers) getCostSyncCoverageReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}
	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	page, err := coldpath.ParseCursorPagination(r, 50, 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	campaignIDs, err := listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{Rows: []map[string]any{}, Freshness: reports.reportFreshness(r.Context())})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()
	rows, total, err := queryCostCoverageRows(chCtx, reports.CHQuery, reports.Pool, customerID, campaignIDs, from, to, page.Limit, page.Offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"campaign_id":      row.CampaignID,
			"clicks":           row.Clicks,
			"spend_micro":      row.SpendMicro,
			"coverage_gap":     row.CoverageGap,
			"network":          row.Network,
			"last_sync_status": row.LastSyncStatus,
		})
	}
	var nextCursor string
	if int64(page.Offset)+int64(len(out)) < total {
		nextCursor = coldpath.EncodeCursor(page.Offset + page.Limit)
	}
	httpresponse.JSON(w, http.StatusOK, ReportRowsResponse{
		Rows:       out,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	})
}

func queryCostCoverageRows(
	ctx context.Context,
	chQuery *database.CHQuery,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]CostCoverageRowDTO, int64, error) {
	if chQuery == nil {
		return nil, 0, fmt.Errorf("clickhouse not configured")
	}
	var total int64
	if err := chQuery.QueryRow(ctx, costCoverageCountQuery, campaignIDs, from, to).Scan(&total); err != nil {
		return nil, 0, err
	}
	chRows, err := chQuery.Query(ctx, costCoverageQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = chRows.Close() }()

	syncStatusByNetwork := map[string]string{}
	if pool != nil {
		syncStatusByNetwork = latestCostSyncStatusByNetwork(ctx, pool, customerID)
	}

	out := make([]CostCoverageRowDTO, 0, limit)
	for chRows.Next() {
		var campaignID uuid.UUID
		var clicks, spendMicro int64
		if err := chRows.Scan(&campaignID, &clicks, &spendMicro); err != nil {
			return nil, 0, err
		}
		row := CostCoverageRowDTO{
			CampaignID:  campaignID.String(),
			Clicks:      clicks,
			SpendMicro:  spendMicro,
			CoverageGap: "missing_cost_snapshots",
		}
		for network, status := range syncStatusByNetwork {
			row.Network = network
			row.LastSyncStatus = status
			break
		}
		out = append(out, row)
	}
	return out, total, chRows.Err()
}

func latestCostSyncStatusByNetwork(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) map[string]string {
	rows, err := db.New(pool).ListCostSyncRuns(ctx, db.ListCostSyncRunsParams{
		Column1: pgtype.UUID{Bytes: customerID, Valid: true},
		Limit:   50,
		Offset:  0,
	})
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		network := row.Network
		if _, seen := out[network]; seen {
			continue
		}
		out[network] = row.Status
	}
	return out
}

func queryCostCoverageExportRows(
	ctx context.Context,
	deps ReportExportDeps,
	customerID uuid.UUID,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]CostCoverageRowDTO, int64, error) {
	return queryCostCoverageRows(ctx, deps.CHQuery, deps.Pool, customerID, campaignIDs, from, to, limit, offset)
}
