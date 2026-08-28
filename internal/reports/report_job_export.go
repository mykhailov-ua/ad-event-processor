package reports

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const reportExportPageSize = 1000

type ReportExportDeps struct {
	Pool                        *pgxpool.Pool
	ClickHouseQuery             *database.ClickHouseQuery
	BuyerPortfolio              BuyerPortfolioReader
	FraudEvidencePackHMACSecret []byte
}

func writeReportCSV(ctx context.Context, deps ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	from, to, err := reportjob.ParseReportRangeFromStrings(spec.From, spec.To)
	if err != nil {
		return err
	}
	if deps.Pool == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	portfolioExport := spec.ReportKey == "campaign-overview" || spec.ReportKey == "customer-portfolio"
	if !portfolioExport && deps.ClickHouseQuery == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	campaignIDs, err := listCustomerCampaignIDs(ctx, deps.Pool, uuid.MustParse(spec.CustomerID))
	if err != nil {
		return err
	}
	if portfolioExport {
		campaignIDs = nil
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	profile := spec.RedactionProfile
	if profile == "" {
		profile = ExportProfileOperatorFull
	}
	if err := writeExportMetaHeader(w, spec.ExportedBy, exportDeploymentID()); err != nil {
		return err
	}
	freshness := DataFreshnessFromClickHouse(ctx, deps.ClickHouseQuery)
	if profile == ExportProfileBuyerSummary && (spec.ReportKey == "customer-fraud-by-type" || spec.ReportKey == "customer-fraud-by-dimension") {
		if err := WriteBuyerFraudExportPreamble(w, freshness); err != nil {
			return err
		}
	}

	switch spec.ReportKey {
	case "placements":
		ivtRates, ierr := QueryPlacementIVTRates(ctx, deps.ClickHouseQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"placement_id", "campaign_id", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr", "ivt_rate"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return QueryPlacementReportRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := ToPlacementReportRowDTO(row, ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)])
				return w.Write([]string{
					dto.PlacementID, dto.CampaignID,
					fmt.Sprintf("%d", dto.Impressions), fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%d", dto.SpendMicro), fmt.Sprintf("%d", dto.RevenueMicro), fmt.Sprintf("%d", dto.ProfitMicro),
					fmt.Sprintf("%.4f", dto.ROIPct), fmt.Sprintf("%.6f", dto.CTR), fmt.Sprintf("%.6f", dto.IVTRate),
				})
			},
		)
	case "keywords":
		ivtRates, ierr := queryKeywordIVTRates(ctx, deps.ClickHouseQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"keyword", "campaign_id", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr", "ivt_rate"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return queryKeywordReportRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := toKeywordReportRowDTO(row, ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)])
				return w.Write([]string{
					dto.Keyword, dto.CampaignID,
					fmt.Sprintf("%d", dto.Impressions), fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%d", dto.SpendMicro), fmt.Sprintf("%d", dto.RevenueMicro), fmt.Sprintf("%d", dto.ProfitMicro),
					fmt.Sprintf("%.4f", dto.ROIPct), fmt.Sprintf("%.6f", dto.CTR), fmt.Sprintf("%.6f", dto.IVTRate),
				})
			},
		)
	case "ivt-by-source":
		if err := w.Write([]string{"campaign_id", "sub1", "sub2", "country", "impressions", "clicks", "ivt_events", "ivt_rate"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]ivtBySourceCHRow, int64, error) {
				return queryIVTBySourceRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row ivtBySourceCHRow) error {
				return w.Write([]string{
					row.CampaignID, row.Sub1, row.Sub2, row.Country,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks),
					fmt.Sprintf("%d", row.IVTEvents), fmt.Sprintf("%.6f", calcIVTRate(row.IVTEvents, row.Clicks)),
				})
			},
		)
	case "traffic-sources":
		if err := w.Write([]string{"channel", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]TrafficSourceRowDTO, int64, error) {
				return queryTrafficSourceRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row TrafficSourceRowDTO) error {
				return w.Write([]string{
					row.Channel,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks), fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.SpendMicro), fmt.Sprintf("%d", row.RevenueMicro), fmt.Sprintf("%d", row.ProfitMicro),
					fmt.Sprintf("%.4f", row.ROIPct), fmt.Sprintf("%.6f", row.CTR),
				})
			},
		)
	case "geo-roi":
		if err := w.Write([]string{"country", "impressions", "clicks", "conversions", "ivt_events", "ivt_rate", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]GeoROIRowDTO, int64, error) {
				return queryGeoROIRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row GeoROIRowDTO) error {
				return w.Write([]string{
					row.Country,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks), fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.IVTEvents), fmt.Sprintf("%.6f", row.IVTRate),
					fmt.Sprintf("%d", row.SpendMicro), fmt.Sprintf("%d", row.RevenueMicro), fmt.Sprintf("%d", row.ProfitMicro),
					fmt.Sprintf("%.4f", row.ROIPct), fmt.Sprintf("%.6f", row.CTR),
				})
			},
		)
	case "telegram":
		if err := w.Write([]string{"start_param", "clicks", "impressions", "conversions", "premium_clicks", "motivated_clicks"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]telegramExportCHRow, int64, error) {
				return queryTelegramExportRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row telegramExportCHRow) error {
				return w.Write([]string{
					row.StartParam,
					fmt.Sprintf("%d", row.Clicks),
					fmt.Sprintf("%d", row.Impressions),
					fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.Premium),
					fmt.Sprintf("%d", row.Motivated),
				})
			},
		)
	case "filter-rejects":
		if err := w.Write([]string{"reject_kind", "reject_count"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]FilterRejectRowDTO, int64, error) {
				return queryFilterRejectRows(ctx, deps.ClickHouseQuery, from, to, limit, offset)
			},
			func(row FilterRejectRowDTO) error {
				return w.Write([]string{row.RejectKind, fmt.Sprintf("%d", row.RejectCount)})
			},
		)
	case "fraud-breakdown":
		fullHeader := []string{"campaign_id", "placement_id", "fraud_reason", "event_count", "silent_reject_count", "silent_reject_ratio"}
		header := ExportColumnsForReport("fraud-breakdown", profile)
		if len(header) == 0 {
			header = fullHeader
		}
		if err := w.Write(header); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]FraudBreakdownRowDTO, int64, error) {
				return queryFraudBreakdownRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row FraudBreakdownRowDTO) error {
				fullRow := []string{
					row.CampaignID, row.PlacementID, row.FraudReason,
					fmt.Sprintf("%d", row.EventCount), fmt.Sprintf("%d", row.SilentRejectCount),
					fmt.Sprintf("%.6f", row.SilentRejectRatio),
				}
				return w.Write(projectExportRow(fullHeader, fullRow, header))
			},
		)
	case "silent-reject-impression-funnel":
		if err := w.Write([]string{
			"campaign_id", "placement_id", "billable_impressions", "silent_reject_impressions",
			"ivt_impressions", "silent_reject_rate", "ivt_impression_rate",
		}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]SilentRejectImpressionFunnelRowDTO, int64, error) {
				return querySilentRejectImpressionFunnelRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row SilentRejectImpressionFunnelRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.PlacementID,
					fmt.Sprintf("%d", row.BillableImpressions), fmt.Sprintf("%d", row.SilentRejectImpressions),
					fmt.Sprintf("%d", row.IVTImpressions),
					fmt.Sprintf("%.6f", row.SilentRejectRate), fmt.Sprintf("%.6f", row.IVTImpressionRate),
				})
			},
		)
	case "rtb-overview":
		if err := w.Write([]string{"deal_id", "bids", "wins", "win_rate", "spend_micro"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]RtbOverviewRowDTO, int64, error) {
				return queryRtbOverviewRows(ctx, deps.ClickHouseQuery, from, to, limit, offset)
			},
			func(row RtbOverviewRowDTO) error {
				return w.Write([]string{
					row.DealID,
					fmt.Sprintf("%d", row.Bids), fmt.Sprintf("%d", row.Wins),
					fmt.Sprintf("%.6f", row.WinRate), fmt.Sprintf("%d", row.SpendMicro),
				})
			},
		)
	case "rtb-no-bid-reasons":
		if err := w.Write([]string{"no_bid_reason", "bid_count"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]RtbNoBidReasonRowDTO, int64, error) {
				return queryRtbNoBidReasonRows(ctx, deps.ClickHouseQuery, from, to, limit, offset)
			},
			func(row RtbNoBidReasonRowDTO) error {
				return w.Write([]string{row.NoBidReason, fmt.Sprintf("%d", row.BidCount)})
			},
		)
	case "pacing-drift":
		if err := w.Write([]string{"campaign_id", "date", "planned_spend_micro", "actual_spend_micro", "drift_pct", "pacing_mode"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]PacingDriftRowDTO, int64, error) {
				return queryPacingDriftExportRows(ctx, deps, campaignIDs, from, to, limit, offset)
			},
			func(row PacingDriftRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.Date,
					fmt.Sprintf("%d", row.PlannedSpendMicro), fmt.Sprintf("%d", row.ActualSpendMicro),
					fmt.Sprintf("%.6f", row.DriftPct), row.PacingMode,
				})
			},
		)
	case "postback-reconciliation":
		fullHeader := []string{"campaign_id", "click_id", "conversion_at", "conversion_value_micro", "ledger_day_fee_micro", "postback_status", "reconcile_status", "error_message"}
		header := ExportColumnsForReport("postback-reconciliation", profile)
		if len(header) == 0 {
			header = fullHeader
		}
		if err := w.Write(header); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]PostbackReconRowDTO, int64, error) {
				return queryPostbackReconExportRows(ctx, deps, campaignIDs, from, to, limit, offset)
			},
			func(row PostbackReconRowDTO) error {
				fullRow := []string{
					row.CampaignID, row.ClickID, row.ConversionAt,
					fmt.Sprintf("%d", row.ConversionValueMicro), fmt.Sprintf("%d", row.LedgerDayFeeMicro),
					row.PostbackStatus, row.ReconcileStatus, row.ErrorMessage,
				}
				return w.Write(projectExportRow(fullHeader, fullRow, header))
			},
		)
	case "spend-velocity":
		err = exportCHMapReport(ctx, w, deps.ClickHouseQuery, campaignIDs, from, to, querySpendVelocityRows,
			[]string{"bucket", "spend_micro", "clicks"},
			"bucket", "spend_micro", "clicks",
		)
	case "daypart-heatmap":
		err = exportCHMapReport(ctx, w, deps.ClickHouseQuery, campaignIDs, from, to, queryDaypartHeatmapRows,
			[]string{"hour", "clicks"},
			"hour", "clicks",
		)
	case "campaign-geo-device":
		err = exportCHMapReport(ctx, w, deps.ClickHouseQuery, campaignIDs, from, to, queryGeoDeviceRows,
			[]string{"country", "device", "clicks"},
			"country", "device", "clicks",
		)
	case "source-quality":
		ivtRates, ierr := QueryPlacementIVTRates(ctx, deps.ClickHouseQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"placement_id", "campaign_id", "clicks", "conversions", "ivt_rate", "roi_pct"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return QueryPlacementReportRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := ToPlacementReportRowDTO(row, ivtRates[ReportMetricsKey(row.Dimension, row.CampaignID)])
				return w.Write([]string{
					dto.PlacementID, dto.CampaignID,
					fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%.6f", dto.IVTRate), fmt.Sprintf("%.4f", dto.ROIPct),
				})
			},
		)
	case "discrepancy-buy-sell":
		err = exportCHMapReport(ctx, w, deps.ClickHouseQuery, campaignIDs, from, to, queryDiscrepancyRows,
			[]string{"campaign_id", "buy_spend_micro", "sell_rev_micro", "delta_micro", "delta_pct"},
			"campaign_id", "buy_spend_micro", "sell_rev_micro", "delta_micro", "delta_pct",
		)
	case "true-roi":
		err = exportCHMapReport(ctx, w, deps.ClickHouseQuery, campaignIDs, from, to, queryTrueROIRows,
			[]string{"campaign_id", "ad_spend_micro", "revenue_micro", "true_profit_micro", "true_roi_pct", "true_cpa_micro", "conversions"},
			"campaign_id", "ad_spend_micro", "revenue_micro", "true_profit_micro", "true_roi_pct", "true_cpa_micro", "conversions",
		)
	case "rtb-geo-device":
		if err := w.Write([]string{"country", "device_os", "bids", "wins", "win_rate", "spend_micro"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]RtbGeoDeviceRowDTO, int64, error) {
				return queryRtbGeoDeviceRows(ctx, deps.ClickHouseQuery, from, to, limit, offset)
			},
			func(row RtbGeoDeviceRowDTO) error {
				return w.Write([]string{
					row.Country, row.DeviceOS,
					fmt.Sprintf("%d", row.Bids), fmt.Sprintf("%d", row.Wins),
					fmt.Sprintf("%.6f", row.WinRate), fmt.Sprintf("%d", row.SpendMicro),
				})
			},
		)
	case "data-quality":
		if err := w.Write([]string{"campaign_id", "date", "postgres_total", "clickhouse_total", "diff_pct", "severity"}); err != nil {
			return err
		}
		customerUUID := uuid.MustParse(spec.CustomerID)
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]DataQualityRowDTO, int64, error) {
				return queryDataQualityExportRows(ctx, deps, customerUUID, campaignIDs, from, to, limit, offset)
			},
			func(row DataQualityRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.Date,
					fmt.Sprintf("%d", row.PostgresTotal), fmt.Sprintf("%d", row.ClickHouseTotal),
					fmt.Sprintf("%.6f", row.DiffPct), row.Severity,
				})
			},
		)
	case "campaign-overview":
		err = writeCampaignOverviewCSV(ctx, w, deps, spec.CustomerID)
	case "customer-portfolio":
		err = writeCustomerPortfolioCSV(ctx, w, deps, spec.CustomerID)
	case "cost-sync-coverage":
		if err := w.Write([]string{"campaign_id", "clicks", "spend_micro", "coverage_gap", "network", "last_sync_status"}); err != nil {
			return err
		}
		customerUUID := uuid.MustParse(spec.CustomerID)
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]CostCoverageRowDTO, int64, error) {
				return queryCostCoverageExportRows(ctx, deps, customerUUID, campaignIDs, from, to, limit, offset)
			},
			func(row CostCoverageRowDTO) error {
				return w.Write([]string{
					row.CampaignID,
					fmt.Sprintf("%d", row.Clicks),
					fmt.Sprintf("%d", row.SpendMicro),
					row.CoverageGap,
					row.Network,
					row.LastSyncStatus,
				})
			},
		)
	case "customer-fraud-by-type":
		rawRows, _, qerr := queryFraudBreakdownRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, 10_000, 0)
		if qerr != nil {
			return qerr
		}
		rows := aggregateCustomerFraudByType(rawRows, "")
		err = writeCustomerFraudByTypeExport(w, profile, rows)
	case "customer-fraud-by-dimension":
		dimRows, _, qerr := buildCustomerFraudByDimensionRows(ctx, deps.ClickHouseQuery, campaignIDs, from, to, "placement", context.Background())
		if qerr != nil {
			return qerr
		}
		err = writeCustomerFraudByDimensionExport(w, profile, dimRows)
	default:
		return fmt.Errorf("unsupported report_key %q", spec.ReportKey)
	}
	if err != nil {
		return err
	}

	w.Flush()
	return w.Error()
}
