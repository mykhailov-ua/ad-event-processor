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
