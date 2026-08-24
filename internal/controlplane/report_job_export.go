package controlplane

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const reportExportPageSize = 1000

type ReportExportDeps struct {
	Pool           *pgxpool.Pool
	CHQuery        *database.CHQuery
	BuyerPortfolio BuyerPortfolioReader
}

func (r *ReportJobRunner) writeReportCSV(ctx context.Context, path string, spec ReportJobSpec) error {
	from, to, err := parseReportRangeFromStrings(spec.From, spec.To)
	if err != nil {
		return err
	}
	if r.deps.Pool == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	portfolioExport := spec.ReportKey == "campaign-overview" || spec.ReportKey == "customer-portfolio"
	if !portfolioExport && r.deps.CHQuery == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	campaignIDs, err := listCustomerCampaignIDs(ctx, r.deps.Pool, uuid.MustParse(spec.CustomerID))
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

	switch spec.ReportKey {
	case "placements":
		ivtRates, ierr := queryPlacementIVTRates(ctx, r.deps.CHQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"placement_id", "campaign_id", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr", "ivt_rate"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return queryPlacementReportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := toPlacementReportRowDTO(row, ivtRates[reportMetricsKey(row.Dimension, row.CampaignID)])
				return w.Write([]string{
					dto.PlacementID, dto.CampaignID,
					fmt.Sprintf("%d", dto.Impressions), fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%d", dto.SpendMicro), fmt.Sprintf("%d", dto.RevenueMicro), fmt.Sprintf("%d", dto.ProfitMicro),
					fmt.Sprintf("%.4f", dto.ROIPct), fmt.Sprintf("%.6f", dto.CTR), fmt.Sprintf("%.6f", dto.IVTRate),
				})
			},
		)
	case "keywords":
		ivtRates, ierr := queryKeywordIVTRates(ctx, r.deps.CHQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"keyword", "campaign_id", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr", "ivt_rate"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return queryKeywordReportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := toKeywordReportRowDTO(row, ivtRates[reportMetricsKey(row.Dimension, row.CampaignID)])
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
				return queryIVTBySourceRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
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
				return queryTrafficSourceRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
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
				return queryGeoROIRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
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
				return queryTelegramExportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
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
				return queryFilterRejectRows(ctx, r.deps.CHQuery, from, to, limit, offset)
			},
			func(row FilterRejectRowDTO) error {
				return w.Write([]string{row.RejectKind, fmt.Sprintf("%d", row.RejectCount)})
			},
		)
	case "fraud-breakdown":
		if err := w.Write([]string{"campaign_id", "placement_id", "fraud_reason", "event_count", "ghost_count", "ghost_ratio"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]FraudBreakdownRowDTO, int64, error) {
				return queryFraudBreakdownRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
			},
			func(row FraudBreakdownRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.PlacementID, row.FraudReason,
					fmt.Sprintf("%d", row.EventCount), fmt.Sprintf("%d", row.GhostCount),
					fmt.Sprintf("%.6f", row.GhostRatio),
				})
			},
		)
	case "ghost-impression-funnel":
		if err := w.Write([]string{
			"campaign_id", "placement_id", "billable_impressions", "ghost_impressions",
			"ivt_impressions", "ghost_rate", "ivt_impression_rate",
		}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]GhostImpressionFunnelRowDTO, int64, error) {
				return queryGhostImpressionFunnelRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
			},
			func(row GhostImpressionFunnelRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.PlacementID,
					fmt.Sprintf("%d", row.BillableImpressions), fmt.Sprintf("%d", row.GhostImpressions),
					fmt.Sprintf("%d", row.IVTImpressions),
					fmt.Sprintf("%.6f", row.GhostRate), fmt.Sprintf("%.6f", row.IVTImpressionRate),
				})
			},
		)
	case "rtb-overview":
		if err := w.Write([]string{"deal_id", "bids", "wins", "win_rate", "spend_micro"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]RtbOverviewRowDTO, int64, error) {
				return queryRtbOverviewRows(ctx, r.deps.CHQuery, from, to, limit, offset)
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
				return queryRtbNoBidReasonRows(ctx, r.deps.CHQuery, from, to, limit, offset)
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
				return queryPacingDriftExportRows(ctx, r.deps, campaignIDs, from, to, limit, offset)
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
		if err := w.Write([]string{"campaign_id", "click_id", "conversion_at", "conversion_value_micro", "ledger_day_fee_micro", "postback_status", "reconcile_status", "error_message"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]PostbackReconRowDTO, int64, error) {
				return queryPostbackReconExportRows(ctx, r.deps, campaignIDs, from, to, limit, offset)
			},
			func(row PostbackReconRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.ClickID, row.ConversionAt,
					fmt.Sprintf("%d", row.ConversionValueMicro), fmt.Sprintf("%d", row.LedgerDayFeeMicro),
					row.PostbackStatus, row.ReconcileStatus, row.ErrorMessage,
				})
			},
		)
	case "spend-velocity":
		err = exportCHMapReport(ctx, w, r.deps.CHQuery, campaignIDs, from, to, querySpendVelocityRows,
			[]string{"bucket", "spend_micro", "clicks"},
			"bucket", "spend_micro", "clicks",
		)
	case "daypart-heatmap":
		err = exportCHMapReport(ctx, w, r.deps.CHQuery, campaignIDs, from, to, queryDaypartHeatmapRows,
			[]string{"hour", "clicks"},
			"hour", "clicks",
		)
	case "campaign-geo-device":
		err = exportCHMapReport(ctx, w, r.deps.CHQuery, campaignIDs, from, to, queryGeoDeviceRows,
			[]string{"country", "device", "clicks"},
			"country", "device", "clicks",
		)
	case "source-quality":
		ivtRates, ierr := queryPlacementIVTRates(ctx, r.deps.CHQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"placement_id", "campaign_id", "clicks", "conversions", "ivt_rate", "roi_pct"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]reportMetricsCHRow, int64, error) {
				return queryPlacementReportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, limit, offset)
			},
			func(row reportMetricsCHRow) error {
				dto := toPlacementReportRowDTO(row, ivtRates[reportMetricsKey(row.Dimension, row.CampaignID)])
				return w.Write([]string{
					dto.PlacementID, dto.CampaignID,
					fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%.6f", dto.IVTRate), fmt.Sprintf("%.4f", dto.ROIPct),
				})
			},
		)
	case "discrepancy-buy-sell":
		err = exportCHMapReport(ctx, w, r.deps.CHQuery, campaignIDs, from, to, queryDiscrepancyRows,
			[]string{"campaign_id", "buy_spend_micro", "sell_rev_micro", "delta_micro", "delta_pct"},
			"campaign_id", "buy_spend_micro", "sell_rev_micro", "delta_micro", "delta_pct",
		)
	case "true-roi":
		err = exportCHMapReport(ctx, w, r.deps.CHQuery, campaignIDs, from, to, queryTrueROIRows,
			[]string{"campaign_id", "ad_spend_micro", "revenue_micro", "true_profit_micro", "true_roi_pct", "true_cpa_micro", "conversions"},
			"campaign_id", "ad_spend_micro", "revenue_micro", "true_profit_micro", "true_roi_pct", "true_cpa_micro", "conversions",
		)
	case "rtb-geo-device":
		if err := w.Write([]string{"country", "device_os", "bids", "wins", "win_rate", "spend_micro"}); err != nil {
			return err
		}
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]RtbGeoDeviceRowDTO, int64, error) {
				return queryRtbGeoDeviceRows(ctx, r.deps.CHQuery, from, to, limit, offset)
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
		if err := w.Write([]string{"campaign_id", "date", "pg_total", "ch_total", "diff_pct", "severity"}); err != nil {
			return err
		}
		customerUUID := uuid.MustParse(spec.CustomerID)
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]DataQualityRowDTO, int64, error) {
				return queryDataQualityExportRows(ctx, r.deps, customerUUID, campaignIDs, from, to, limit, offset)
			},
			func(row DataQualityRowDTO) error {
				return w.Write([]string{
					row.CampaignID, row.Date,
					fmt.Sprintf("%d", row.PGTotal), fmt.Sprintf("%d", row.CHTotal),
					fmt.Sprintf("%.6f", row.DiffPct), row.Severity,
				})
			},
		)
	case "campaign-overview":
		err = writeCampaignOverviewCSV(ctx, w, r.deps, spec.CustomerID)
	case "customer-portfolio":
		err = writeCustomerPortfolioCSV(ctx, w, r.deps, spec.CustomerID)
	case "cost-sync-coverage":
		if err := w.Write([]string{"campaign_id", "clicks", "spend_micro", "coverage_gap", "network", "last_sync_status"}); err != nil {
			return err
		}
		customerUUID := uuid.MustParse(spec.CustomerID)
		err = paginateCHExport(reportExportPageSize,
			func(offset, limit int) ([]CostCoverageRowDTO, int64, error) {
				return queryCostCoverageExportRows(ctx, r.deps, customerUUID, campaignIDs, from, to, limit, offset)
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
	default:
		return fmt.Errorf("unsupported report_key %q", spec.ReportKey)
	}
	if err != nil {
		return err
	}

	w.Flush()
	return w.Error()
}
