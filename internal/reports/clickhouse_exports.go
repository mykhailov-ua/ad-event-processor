package reports

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reports/clickhouse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultReportLookback = 7 * 24 * time.Hour

var reportClickHouseQueryTimeout = clickhouse.ReportClickHouseQueryTimeout()

func ParseReportRange(r *http.Request) (from, to time.Time, err error) {
	return clickhouse.ParseReportRange(r)
}

func ListCustomerCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	return clickhouse.ListCustomerCampaignIDs(ctx, pool, customerID)
}

func ReportClickHouseQueryTimeout() time.Duration {
	return clickhouse.ReportClickHouseQueryTimeout()
}

func listCustomerCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	return ListCustomerCampaignIDs(ctx, pool, customerID)
}

func ClickHouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return clickhouse.ClickHouseQueryContext(ctx)
}

func QueryPlacementReportRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reportMetricsCHRow, int64, error) {
	rows, total, err := clickhouse.QueryPlacementReportRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return reportMetricsCHRowsFromClickhouse(rows), total, nil
}

func QueryPlacementIVTRates(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]float64, error) {
	return clickhouse.QueryPlacementIVTRates(ctx, clickhouseQuery, campaignIDs, from, to)
}

func QueryCampaignEconomicsCH(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
) (clickhouse.CampaignEconomicsCH, error) {
	return clickhouse.QueryCampaignEconomicsCH(ctx, clickhouseQuery, campaignID, from, to)
}

func QueryClickHouseCampaignDailyEventTotals(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
) (map[string]uint64, error) {
	return clickhouse.QueryClickHouseCampaignDailyEventTotals(ctx, clickhouseQuery, campaignIDs, from, to)
}

func reportMetricsCHRowsFromClickhouse(rows []clickhouse.ReportMetricsCHRow) []reportMetricsCHRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]reportMetricsCHRow, len(rows))
	for i := range rows {
		out[i] = reportMetricsCHRow(rows[i])
	}
	return out
}

func CampaignDailyTotalKey(campaignID uuid.UUID, day time.Time) string {
	return clickhouse.CampaignDailyTotalKey(campaignID, day)
}

func querySpendVelocityRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QuerySpendVelocityRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryTrueROIRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryTrueROIRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryDaypartHeatmapRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryDaypartHeatmapRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryGeoDeviceRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryGeoDeviceRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryDiscrepancyRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryDiscrepancyRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryKeywordIVTRates(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time) (map[string]float64, error) {
	return clickhouse.QueryKeywordIVTRates(ctx, clickhouseQuery, campaignIDs, from, to)
}

func queryKeywordReportRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]reportMetricsCHRow, int64, error) {
	rows, total, err := clickhouse.QueryKeywordReportRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return reportMetricsCHRowsFromClickhouse(rows), total, nil
}
