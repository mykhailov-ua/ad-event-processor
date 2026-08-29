package reports

import (
	"context"
	"errors"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports/clickhouse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errReportExportUnwired = errors.New("report export not wired")

type ReportExportDeps struct {
	Pool                        *pgxpool.Pool
	ClickHouseQuery             *database.ClickHouseQuery
	BuyerPortfolio              BuyerPortfolioReader
	FraudEvidencePackHMACSecret []byte
}

var (
	ExportActorLabel   func(context.Context) string
	ExportDeploymentID func() string
)

var reportExportWrite func(context.Context, ReportExportDeps, string, reportjob.ReportJobSpec) error

func SetReportExportWrite(fn func(context.Context, ReportExportDeps, string, reportjob.ReportJobSpec) error) {
	reportExportWrite = fn
}

func WriteReportExport(ctx context.Context, deps ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	if reportExportWrite == nil {
		return errReportExportUnwired
	}
	return reportExportWrite(ctx, deps, path, spec)
}

type ReportMetricsCHRow = reportMetricsCHRow

type TelegramExportCHRow = clickhouse.TelegramExportCHRow

func QueryKeywordReportRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]ReportMetricsCHRow, int64, error) {
	rows, total, err := clickhouse.QueryKeywordReportRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return reportMetricsCHRowsFromClickhouse(rows), total, nil
}

func QueryKeywordIVTRates(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time) (map[string]float64, error) {
	return clickhouse.QueryKeywordIVTRates(ctx, clickhouseQuery, campaignIDs, from, to)
}

func ToKeywordReportRowDTO(row ReportMetricsCHRow, ivtRate float64) KeywordReportRowDTO {
	return toKeywordReportRowDTO(row, ivtRate)
}

func QueryTrafficSourceRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]TrafficSourceRowDTO, int64, error) {
	return queryTrafficSourceRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryGeoROIRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]GeoROIRowDTO, int64, error) {
	return queryGeoROIRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryTelegramExportRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]TelegramExportCHRow, int64, error) {
	return clickhouse.QueryTelegramExportRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryFilterRejectRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]FilterRejectRowDTO, int64, error) {
	return queryFilterRejectRows(ctx, clickhouseQuery, from, to, limit, offset)
}

func QueryRtbOverviewRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]RtbOverviewRowDTO, int64, error) {
	return queryRtbOverviewRows(ctx, clickhouseQuery, from, to, limit, offset)
}

func QueryRtbNoBidReasonRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]RtbNoBidReasonRowDTO, int64, error) {
	return queryRtbNoBidReasonRows(ctx, clickhouseQuery, from, to, limit, offset)
}

func QueryRtbGeoDeviceRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]RtbGeoDeviceRowDTO, int64, error) {
	return queryRtbGeoDeviceRows(ctx, clickhouseQuery, from, to, limit, offset)
}

func QueryPacingDriftExportRows(ctx context.Context, deps ReportExportDeps, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]PacingDriftRowDTO, int64, error) {
	return queryPacingDriftExportRows(ctx, deps, campaignIDs, from, to, limit, offset)
}

func QueryPostbackReconExportRows(ctx context.Context, deps ReportExportDeps, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]PostbackReconRowDTO, int64, error) {
	return queryPostbackReconExportRows(ctx, deps, campaignIDs, from, to, limit, offset)
}

func QuerySpendVelocityRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QuerySpendVelocityRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryDaypartHeatmapRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryDaypartHeatmapRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryGeoDeviceRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryGeoDeviceRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryDiscrepancyRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryDiscrepancyRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryTrueROIRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return clickhouse.QueryTrueROIRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryDataQualityExportRows(ctx context.Context, deps ReportExportDeps, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]DataQualityRowDTO, int64, error) {
	return queryDataQualityExportRows(ctx, deps, customerID, campaignIDs, from, to, limit, offset)
}

func QueryCostCoverageExportRows(ctx context.Context, deps ReportExportDeps, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]CostCoverageRowDTO, int64, error) {
	return queryCostCoverageExportRows(ctx, deps, customerID, campaignIDs, from, to, limit, offset)
}
