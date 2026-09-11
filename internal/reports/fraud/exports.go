package fraud

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func fraudExportAPI() reports.FraudExportAPI {
	return reports.FraudExportAPI{
		ReportPermsFraudCustomer:              ReportPermsFraudCustomer,
		ReportPermsCustomerFraudEvidence:      reportPermsCustomerFraudEvidence,
		FraudReasonToCategory:                 FraudReasonToCategory,
		BuildSignedFraudEvidencePack:          BuildSignedFraudEvidencePack,
		VerifyFraudEvidencePackSignature:      VerifyFraudEvidencePackSignature,
		ScrubCustomerFraudEvidencePack:        ScrubCustomerFraudEvidencePack,
		BuildCustomerFraudOverview:            BuildCustomerFraudOverview,
		AttachInvalidSpendKPI:                 AttachInvalidSpendKPI,
		QueryCustomerFraudOverview:            QueryCustomerFraudOverview,
		QueryCustomerFraudDailySeries:         QueryCustomerFraudDailySeries,
		ComputeAttributionCoverage:            ComputeAttributionCoverage,
		QueryWorstIVTSources:                  QueryWorstIVTSources,
		QueryWorstIVTCountries:                QueryWorstIVTCountries,
		QueryFraudBreakdownRows:               queryFraudBreakdownRows,
		QueryIVTBySourceRows:                  QueryIVTBySourceRows,
		QuerySilentRejectImpressionFunnelRows: querySilentRejectImpressionFunnelRows,
		AggregateCustomerFraudByType:          aggregateCustomerFraudByType,
		BuildCustomerFraudByDimensionRows:     buildCustomerFraudByDimensionRows,
		WriteFraudEvidencePackBulkZip:         WriteFraudEvidencePackBulkZip,
		QueryFraudEvidencePackFraudCH:         queryFraudEvidencePackFraudCH,
		AggregateFraudEvidenceSignals:         aggregateFraudEvidenceSignals,
		UpsertMLShadowDeltaSnapshot:           UpsertMLShadowDeltaSnapshot,
		LoadMLShadowDeltaSnapshot:             LoadMLShadowDeltaSnapshot,
		MLShadowDeltaSnapshotFreshness:        MLShadowDeltaSnapshotFreshness,
		PaginateMLShadowDeltaSnapshotRows:     PaginateMLShadowDeltaSnapshotRows,
		QueryMLShadowDeltaRows:                QueryMLShadowDeltaRows,
	}
}

func WriteFraudEvidencePackBulkZip(ctx context.Context, deps reports.ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	return writeFraudEvidencePackBulkZip(ctx, deps, path, spec)
}

func QueryFraudBreakdownRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reports.FraudBreakdownRowDTO, int64, error) {
	return queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryIVTBySourceRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reports.IVTBySourceRowDTO, int64, error) {
	raw, total, err := queryIVTBySourceRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]reports.IVTBySourceRowDTO, 0, len(raw))
	for _, row := range raw {
		rows = append(rows, reports.IVTBySourceRowDTO{
			CampaignID:  row.CampaignID,
			Sub1:        row.Sub1,
			Sub2:        row.Sub2,
			Country:     row.Country,
			Impressions: row.Impressions,
			Clicks:      row.Clicks,
			IVTEvents:   row.IVTEvents,
			IVTRate:     reports.CalcIVTRate(row.IVTEvents, row.Clicks),
		})
	}
	return rows, total, nil
}

func AggregateCustomerFraudByType(rows []reports.FraudBreakdownRowDTO, categoryFilter string) []reports.CustomerFraudByTypeRowDTO {
	return aggregateCustomerFraudByType(rows, categoryFilter)
}

func BuildCustomerFraudByDimensionRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	dimension string,
	scrubCtx context.Context,
) ([]reports.CustomerFraudByDimensionRowDTO, bool, error) {
	return buildCustomerFraudByDimensionRows(ctx, clickhouseQuery, campaignIDs, from, to, dimension, scrubCtx)
}

func QuerySilentRejectImpressionFunnelRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
) ([]reports.SilentRejectImpressionFunnelRowDTO, int64, error) {
	return querySilentRejectImpressionFunnelRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func QueryWireSignalBreakdownRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
	scrubCtx context.Context,
) ([]reports.WireSignalBreakdownRowDTO, int64, error) {
	return queryWireSignalBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset, scrubCtx)
}

func QuerySignalEffectivenessRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	limit, offset int,
	scrubCtx context.Context,
) ([]reports.SignalEffectivenessRowDTO, int64, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, 0, nil
	}
	rawRows, _, err := queryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, 10_000, 0)
	if err != nil {
		return nil, 0, err
	}
	rows := aggregateSignalEffectiveness(rawRows, scrubCtx)
	total := int64(len(rows))
	if offset >= len(rows) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], total, nil
}

func QueryLayerDesyncDrilldownRows(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	minDesync uint8,
	limit, offset int,
	scrubCtx context.Context,
) ([]reports.LayerDesyncDrilldownRowDTO, int64, error) {
	rows, total, _, err := queryLayerDesyncDrilldown(ctx, clickhouseQuery, campaignIDs, from, to, minDesync, limit, offset, scrubCtx)
	return rows, total, err
}

func QueryLayerDesyncDrilldownSeries(
	ctx context.Context,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	minDesync uint8,
) ([]reports.LayerDesyncDrilldownSeriesPointDTO, error) {
	if clickhouseQuery == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	seriesRows, err := clickhouseQuery.Query(ctx, layerDesyncDrilldownSeriesQuery, campaignIDs, from, to, minDesync, reports.MaxChartSeriesPoints)
	if err != nil {
		return nil, err
	}
	defer func() { _ = seriesRows.Close() }()
	series := make([]reports.LayerDesyncDrilldownSeriesPointDTO, 0, 48)
	for seriesRows.Next() {
		var bucket time.Time
		var events, silent int64
		if err := seriesRows.Scan(&bucket, &events, &silent); err != nil {
			return nil, err
		}
		series = append(series, reports.LayerDesyncDrilldownSeriesPointDTO{
			Label:             bucket.UTC().Format(time.RFC3339),
			EventCount:        events,
			SilentRejectCount: silent,
		})
	}
	return series, seriesRows.Err()
}
