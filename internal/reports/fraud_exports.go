package reports

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var fraudRegistrar func(*ReportsHTTPHandlers, *http.ServeMux)

func SetFraudRegistrar(fn func(*ReportsHTTPHandlers, *http.ServeMux)) {
	fraudRegistrar = fn
}

func registerFraudReports(h *ReportsHTTPHandlers, mux *http.ServeMux) {
	if fraudRegistrar != nil {
		fraudRegistrar(h, mux)
	}
}

type FraudExportAPI struct {
	ReportPermsFraudCustomer              func() []string
	ReportPermsCustomerFraudEvidence      func() []string
	FraudReasonToCategory                 func(string) (string, string)
	BuildSignedFraudEvidencePack          func([]byte, FraudEvidencePackDTO) (FraudEvidencePackDTO, error)
	VerifyFraudEvidencePackSignature      func([]byte, FraudEvidencePackDTO) error
	ScrubCustomerFraudEvidencePack        func(FraudEvidencePackDTO) FraudEvidencePackDTO
	BuildCustomerFraudOverview            func(int64, int64, int64, DataFreshnessDTO) CustomerFraudOverviewDTO
	AttachInvalidSpendKPI                 func(*CustomerFraudOverviewDTO, int64, int64, int64, int64, float64)
	QueryCustomerFraudOverview            func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time) (int64, int64, int64, error)
	QueryCustomerFraudDailySeries         func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time) ([]CustomerFraudSeriesPointDTO, error)
	ComputeAttributionCoverage            func(int64, int64) float64
	QueryWorstIVTSources                  func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, int) ([]SourceRowDTO, error)
	QueryWorstIVTCountries                func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, int) ([]FraudGeoHintDTO, error)
	QueryFraudBreakdownRows               func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, int, int) ([]FraudBreakdownRowDTO, int64, error)
	QueryIVTBySourceRows                  func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, int, int) ([]IVTBySourceRowDTO, int64, error)
	QuerySilentRejectImpressionFunnelRows func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, int, int) ([]SilentRejectImpressionFunnelRowDTO, int64, error)
	AggregateCustomerFraudByType          func([]FraudBreakdownRowDTO, string) []CustomerFraudByTypeRowDTO
	BuildCustomerFraudByDimensionRows     func(context.Context, *database.ClickHouseQuery, []uuid.UUID, time.Time, time.Time, string, context.Context) ([]CustomerFraudByDimensionRowDTO, bool, error)
	WriteFraudEvidencePackBulkZip         func(context.Context, ReportExportDeps, string, reportjob.ReportJobSpec) error
	QueryFraudEvidencePackFraudCH         func(context.Context, *database.ClickHouseQuery, []uuid.UUID, string, time.Time, time.Time) ([]FraudEvidenceFraudRowDTO, error)
	AggregateFraudEvidenceSignals         func([]FraudEvidenceFraudRowDTO) FraudEvidenceSignalsDTO
	UpsertMLShadowDeltaSnapshot           func(context.Context, *pgxpool.Pool, MLShadowDeltaSnapshot) error
	LoadMLShadowDeltaSnapshot             func(context.Context, *pgxpool.Pool) (MLShadowDeltaSnapshot, bool, error)
	MLShadowDeltaSnapshotFreshness        func(MLShadowDeltaSnapshot, time.Time) DataFreshnessDTO
	PaginateMLShadowDeltaSnapshotRows     func([]map[string]any, int, int) ([]map[string]any, int64)
	QueryMLShadowDeltaRows                func(context.Context, *database.ClickHouseQuery, time.Time, time.Time, int, int) ([]map[string]any, int64, error)
}

var fraudExports FraudExportAPI

func SetFraudExports(api FraudExportAPI) {
	fraudExports = api
}

func ReportPermsCustomerFraudEvidence() []string {
	if fraudExports.ReportPermsCustomerFraudEvidence != nil {
		return fraudExports.ReportPermsCustomerFraudEvidence()
	}
	return []string{"campaigns:read"}
}

var (
	reportPermsCampaignRead  = []string{"campaigns:read", permCampaignsReadMasked}
	reportPermsFraudOperator = []string{"audit:read", "campaigns:read"}
)

func ReportPermsFraudCustomer() []string {
	if fraudExports.ReportPermsFraudCustomer != nil {
		return fraudExports.ReportPermsFraudCustomer()
	}
	return []string{"audit:read", "campaigns:read", "campaigns:read:masked"}
}

func FraudReasonToCategory(reason string) (string, string) {
	if fraudExports.FraudReasonToCategory != nil {
		return fraudExports.FraudReasonToCategory(reason)
	}
	return "unknown", "Unknown"
}

func BuildSignedFraudEvidencePack(secret []byte, pack FraudEvidencePackDTO) (FraudEvidencePackDTO, error) {
	return fraudExports.BuildSignedFraudEvidencePack(secret, pack)
}

func VerifyFraudEvidencePackSignature(secret []byte, pack FraudEvidencePackDTO) error {
	return fraudExports.VerifyFraudEvidencePackSignature(secret, pack)
}

func ScrubCustomerFraudEvidencePack(pack FraudEvidencePackDTO) FraudEvidencePackDTO {
	return fraudExports.ScrubCustomerFraudEvidencePack(pack)
}

func BuildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents int64, freshness DataFreshnessDTO) CustomerFraudOverviewDTO {
	return fraudExports.BuildCustomerFraudOverview(totalEvents, blockedEvents, silentRejectEvents, freshness)
}

func AttachInvalidSpendKPI(out *CustomerFraudOverviewDTO, blockedEvents, silentRejectEvents, totalEvents int64, spendMicros int64, attributionCoverage float64) {
	fraudExports.AttachInvalidSpendKPI(out, blockedEvents, silentRejectEvents, totalEvents, spendMicros, attributionCoverage)
}

func QueryCustomerFraudOverview(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time) (int64, int64, int64, error) {
	return fraudExports.QueryCustomerFraudOverview(ctx, clickhouseQuery, campaignIDs, from, to)
}

func QueryCustomerFraudDailySeries(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time) ([]CustomerFraudSeriesPointDTO, error) {
	return fraudExports.QueryCustomerFraudDailySeries(ctx, clickhouseQuery, campaignIDs, from, to)
}

func ComputeAttributionCoverage(totalEvents, attributedEvents int64) float64 {
	return fraudExports.ComputeAttributionCoverage(totalEvents, attributedEvents)
}

func QueryWorstIVTSources(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]SourceRowDTO, error) {
	return fraudExports.QueryWorstIVTSources(ctx, clickhouseQuery, campaignIDs, from, to, limit)
}

func QueryWorstIVTCountries(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]FraudGeoHintDTO, error) {
	return fraudExports.QueryWorstIVTCountries(ctx, clickhouseQuery, campaignIDs, from, to, limit)
}

func UpsertMLShadowDeltaSnapshot(ctx context.Context, pool *pgxpool.Pool, snap MLShadowDeltaSnapshot) error {
	return fraudExports.UpsertMLShadowDeltaSnapshot(ctx, pool, snap)
}

func LoadMLShadowDeltaSnapshot(ctx context.Context, pool *pgxpool.Pool) (MLShadowDeltaSnapshot, bool, error) {
	return fraudExports.LoadMLShadowDeltaSnapshot(ctx, pool)
}

func MLShadowDeltaSnapshotFreshness(snap MLShadowDeltaSnapshot, now time.Time) DataFreshnessDTO {
	return fraudExports.MLShadowDeltaSnapshotFreshness(snap, now)
}

func PaginateMLShadowDeltaSnapshotRows(rows []map[string]any, limit, offset int) ([]map[string]any, int64) {
	return fraudExports.PaginateMLShadowDeltaSnapshotRows(rows, limit, offset)
}

func QueryMLShadowDeltaRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, limit, offset int) ([]map[string]any, int64, error) {
	return fraudExports.QueryMLShadowDeltaRows(ctx, clickhouseQuery, from, to, limit, offset)
}

func CalcSilentRejectRatio(silentRejectCount, eventCount int64) float64 {
	if eventCount <= 0 {
		return 0
	}
	return float64(silentRejectCount) / float64(eventCount)
}
