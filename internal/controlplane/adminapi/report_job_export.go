package adminapi

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const reportExportPageSize = 1000

type ReportExportDeps struct {
	Pool    *pgxpool.Pool
	CHQuery *database.CHQuery
}

func (r *ReportJobRunner) writeReportCSV(ctx context.Context, path string, spec ReportJobSpec) error {
	from, to, err := parseReportRangeFromStrings(spec.From, spec.To)
	if err != nil {
		return err
	}
	if r.deps.Pool == nil || r.deps.CHQuery == nil {
		return fmt.Errorf("report export dependencies not configured")
	}
	campaignIDs, err := listCustomerCampaignIDs(ctx, r.deps.Pool, uuid.MustParse(spec.CustomerID))
	if err != nil {
		return err
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
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryPlacementReportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				dto := toPlacementReportRowDTO(row, ivtRates[placementRowKey(row.PlacementID, row.CampaignID)])
				if err := w.Write([]string{
					dto.PlacementID, dto.CampaignID,
					fmt.Sprintf("%d", dto.Impressions), fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%d", dto.SpendMicro), fmt.Sprintf("%d", dto.RevenueMicro), fmt.Sprintf("%d", dto.ProfitMicro),
					fmt.Sprintf("%.4f", dto.ROIPct), fmt.Sprintf("%.6f", dto.CTR), fmt.Sprintf("%.6f", dto.IVTRate),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	case "keywords":
		ivtRates, ierr := queryKeywordIVTRates(ctx, r.deps.CHQuery, campaignIDs, from, to)
		if ierr != nil {
			return ierr
		}
		if err := w.Write([]string{"keyword", "campaign_id", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr", "ivt_rate"}); err != nil {
			return err
		}
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryKeywordReportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				dto := toKeywordReportRowDTO(row, ivtRates[keywordRowKey(row.Keyword, row.CampaignID)])
				if err := w.Write([]string{
					dto.Keyword, dto.CampaignID,
					fmt.Sprintf("%d", dto.Impressions), fmt.Sprintf("%d", dto.Clicks), fmt.Sprintf("%d", dto.Conversions),
					fmt.Sprintf("%d", dto.SpendMicro), fmt.Sprintf("%d", dto.RevenueMicro), fmt.Sprintf("%d", dto.ProfitMicro),
					fmt.Sprintf("%.4f", dto.ROIPct), fmt.Sprintf("%.6f", dto.CTR), fmt.Sprintf("%.6f", dto.IVTRate),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	case "ivt-by-source":
		if err := w.Write([]string{"campaign_id", "sub1", "sub2", "country", "impressions", "clicks", "ivt_events", "ivt_rate"}); err != nil {
			return err
		}
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryIVTBySourceRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				if err := w.Write([]string{
					row.CampaignID, row.Sub1, row.Sub2, row.Country,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks),
					fmt.Sprintf("%d", row.IVTEvents), fmt.Sprintf("%.6f", calcIVTRate(row.IVTEvents, row.Clicks)),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	case "traffic-sources":
		if err := w.Write([]string{"channel", "impressions", "clicks", "conversions", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr"}); err != nil {
			return err
		}
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryTrafficSourceRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				profit := row.RevenueMicro - row.SpendMicro
				if err := w.Write([]string{
					row.Channel,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks), fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.SpendMicro), fmt.Sprintf("%d", row.RevenueMicro), fmt.Sprintf("%d", profit),
					fmt.Sprintf("%.4f", calcROIPct(profit, row.SpendMicro)), fmt.Sprintf("%.6f", calcCTR(row.Clicks, row.Impressions)),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	case "geo-roi":
		if err := w.Write([]string{"country", "impressions", "clicks", "conversions", "ivt_events", "ivt_rate", "spend_micro", "revenue_micro", "profit_micro", "roi_pct", "ctr"}); err != nil {
			return err
		}
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryGeoROIRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				if err := w.Write([]string{
					row.Country,
					fmt.Sprintf("%d", row.Impressions), fmt.Sprintf("%d", row.Clicks), fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.IVTEvents), fmt.Sprintf("%.6f", row.IVTRate),
					fmt.Sprintf("%d", row.SpendMicro), fmt.Sprintf("%d", row.RevenueMicro), fmt.Sprintf("%d", row.ProfitMicro),
					fmt.Sprintf("%.4f", row.ROIPct), fmt.Sprintf("%.6f", row.CTR),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	case "telegram":
		if err := w.Write([]string{"start_param", "clicks", "impressions", "conversions", "premium_clicks", "motivated_clicks"}); err != nil {
			return err
		}
		for offset := 0; ; offset += reportExportPageSize {
			rows, total, qerr := queryTelegramExportRows(ctx, r.deps.CHQuery, campaignIDs, from, to, reportExportPageSize, offset)
			if qerr != nil {
				return qerr
			}
			for _, row := range rows {
				if err := w.Write([]string{
					row.StartParam,
					fmt.Sprintf("%d", row.Clicks),
					fmt.Sprintf("%d", row.Impressions),
					fmt.Sprintf("%d", row.Conversions),
					fmt.Sprintf("%d", row.Premium),
					fmt.Sprintf("%d", row.Motivated),
				}); err != nil {
					return err
				}
			}
			if int64(offset+len(rows)) >= total || len(rows) == 0 {
				break
			}
		}
	default:
		return fmt.Errorf("unsupported report_key %q", spec.ReportKey)
	}

	w.Flush()
	return w.Error()
}
