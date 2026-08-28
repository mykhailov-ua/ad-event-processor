package reports

import (
	"context"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	libreports "ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

type (
	DataFreshnessDTO = libreports.DataFreshnessDTO
	SourceRowDTO     = libreports.SourceRowDTO
	FraudGeoHintDTO  = libreports.FraudGeoHintDTO
	ReportExportDeps = libreports.ReportExportDeps
)

type FraudBreakdownRowDTO struct {
	CampaignID         string  `json:"campaign_id"`
	PlacementID        string  `json:"placement_id,omitempty"`
	FraudReason        string  `json:"fraud_reason,omitempty"`
	FraudCategory      string  `json:"fraud_category,omitempty"`
	FraudCategoryLabel string  `json:"fraud_category_label,omitempty"`
	EventCount         int64   `json:"event_count"`
	SilentRejectCount  int64   `json:"silent_reject_count"`
	SilentRejectRatio  float64 `json:"silent_reject_ratio"`
}

type FraudEvidenceTimelineRowDTO struct {
	EventType   string `json:"event_type"`
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	Country     string `json:"country,omitempty"`
	Sub1        string `json:"sub1,omitempty"`
}

type FraudEvidenceFraudRowDTO struct {
	EventType         string `json:"event_type"`
	CampaignID        string `json:"campaign_id"`
	PlacementID       string `json:"placement_id,omitempty"`
	FraudReason       string `json:"fraud_reason"`
	FraudScore        uint32 `json:"fraud_score"`
	LayerDesyncCount  uint8  `json:"layer_desync_count"`
	SilentRejectEvent bool   `json:"silent_reject_event"`
	CreatedAt         string `json:"created_at"`
}

type FraudEvidenceSignalsDTO struct {
	FraudReasons        []string `json:"fraud_reasons"`
	MaxFraudScore       uint32   `json:"max_fraud_score"`
	MaxLayerDesyncCount uint8    `json:"max_layer_desync_count"`
	SilentRejectEvents  int      `json:"silent_reject_events"`
}

type FraudEvidencePackDTO struct {
	Version      string                        `json:"version"`
	ClickID      string                        `json:"click_id"`
	CustomerID   string                        `json:"customer_id"`
	CampaignID   string                        `json:"campaign_id,omitempty"`
	GeneratedAt  string                        `json:"generated_at"`
	RangeFrom    string                        `json:"range_from"`
	RangeTo      string                        `json:"range_to"`
	Timeline     []FraudEvidenceTimelineRowDTO `json:"timeline"`
	FraudEvents  []FraudEvidenceFraudRowDTO    `json:"fraud_events"`
	Signals      FraudEvidenceSignalsDTO       `json:"signals"`
	DigestSHA256 string                        `json:"digest_sha256"`
	Signature    string                        `json:"signature"`
}

type CustomerFraudSeriesPointDTO struct {
	Label              string `json:"label"`
	BlockedEvents      int64  `json:"blocked_events"`
	SilentRejectEvents int64  `json:"silent_reject_events"`
	IVTEvents          int64  `json:"ivt_events,omitempty"`
}

type CustomerFraudOverviewDTO struct {
	TotalEvents             int64                         `json:"total_events"`
	BlockedEvents           int64                         `json:"blocked_events"`
	SilentRejectEvents      int64                         `json:"silent_reject_events"`
	BlockRate               float64                       `json:"block_rate"`
	BlockRateDisplay        string                        `json:"block_rate_display"`
	SilentRejectRate        float64                       `json:"silent_reject_rate"`
	SilentRejectRateDisplay string                        `json:"silent_reject_rate_display"`
	IVTRate                 float64                       `json:"ivt_rate,omitempty"`
	IVTRateDisplay          string                        `json:"ivt_rate_display,omitempty"`
	Freshness               DataFreshnessDTO              `json:"freshness"`
	Series                  []CustomerFraudSeriesPointDTO `json:"series,omitempty"`
	Disclaimer              string                        `json:"disclaimer,omitempty"`
	InvalidSpendMicros      int64                         `json:"invalid_spend_micros,omitempty"`
	InvalidSpendDisplay     string                        `json:"invalid_spend_display,omitempty"`
	InvalidSpendSharePct    float64                       `json:"invalid_spend_share_pct,omitempty"`
	ShareLabel              string                        `json:"share_label,omitempty"`
}

type CustomerFraudByTypeRowDTO struct {
	CampaignID         string  `json:"campaign_id"`
	FraudCategory      string  `json:"fraud_category"`
	FraudCategoryLabel string  `json:"fraud_category_label"`
	EventCount         int64   `json:"event_count"`
	SilentRejectCount  int64   `json:"silent_reject_count"`
	SharePct           float64 `json:"share_pct"`
	ShareLabel         string  `json:"share_label"`
	SilentRejectRatio  float64 `json:"silent_reject_ratio"`
}

type CustomerFraudByDimensionRowDTO struct {
	DimensionValue        string  `json:"dimension_value"`
	CampaignID            string  `json:"campaign_id,omitempty"`
	Impressions           int64   `json:"impressions"`
	Clicks                int64   `json:"clicks"`
	IVTEvents             int64   `json:"ivt_events"`
	BlockedEvents         int64   `json:"blocked_events"`
	IVTRate               float64 `json:"ivt_rate"`
	IVTRateLabel          string  `json:"ivt_rate_label"`
	TopFraudCategory      string  `json:"top_fraud_category,omitempty"`
	TopFraudCategoryLabel string  `json:"top_fraud_category_label,omitempty"`
	DeltaLabel            string  `json:"delta_label,omitempty"`
	DeltaTone             string  `json:"delta_tone,omitempty"`
}

type SilentRejectImpressionFunnelRowDTO struct {
	CampaignID              string  `json:"campaign_id"`
	PlacementID             string  `json:"placement_id,omitempty"`
	BillableImpressions     int64   `json:"billable_impressions"`
	SilentRejectImpressions int64   `json:"silent_reject_impressions"`
	IVTImpressions          int64   `json:"ivt_impressions"`
	SilentRejectRate        float64 `json:"silent_reject_rate"`
	IVTImpressionRate       float64 `json:"ivt_impression_rate"`
}

type IVTBySourceRowDTO struct {
	CampaignID  string  `json:"campaign_id"`
	Sub1        string  `json:"sub1,omitempty"`
	Sub2        string  `json:"sub2,omitempty"`
	Country     string  `json:"country,omitempty"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	IVTEvents   int64   `json:"ivt_events"`
	IVTRate     float64 `json:"ivt_rate"`
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
}

var fraudExports FraudExportAPI

func SetFraudExports(api FraudExportAPI) {
	fraudExports = api
}

func ReportPermsFraudCustomer() []string {
	if fraudExports.ReportPermsFraudCustomer != nil {
		return fraudExports.ReportPermsFraudCustomer()
	}
	return []string{"audit:read", "campaigns:read", "campaigns:read:masked"}
}

func reportPermsCustomerFraudEvidence() []string {
	if fraudExports.ReportPermsCustomerFraudEvidence != nil {
		return fraudExports.ReportPermsCustomerFraudEvidence()
	}
	return []string{"campaigns:read"}
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

func queryFraudBreakdownRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]FraudBreakdownRowDTO, int64, error) {
	return fraudExports.QueryFraudBreakdownRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func queryIVTBySourceRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]IVTBySourceRowDTO, int64, error) {
	return fraudExports.QueryIVTBySourceRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func querySilentRejectImpressionFunnelRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]SilentRejectImpressionFunnelRowDTO, int64, error) {
	return fraudExports.QuerySilentRejectImpressionFunnelRows(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
}

func aggregateCustomerFraudByType(rows []FraudBreakdownRowDTO, categoryFilter string) []CustomerFraudByTypeRowDTO {
	return fraudExports.AggregateCustomerFraudByType(rows, categoryFilter)
}

func buildCustomerFraudByDimensionRows(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, dimension string, scrubCtx context.Context) ([]CustomerFraudByDimensionRowDTO, bool, error) {
	return fraudExports.BuildCustomerFraudByDimensionRows(ctx, clickhouseQuery, campaignIDs, from, to, dimension, scrubCtx)
}

func writeFraudEvidencePackBulkZip(ctx context.Context, deps ReportExportDeps, path string, spec reportjob.ReportJobSpec) error {
	return fraudExports.WriteFraudEvidencePackBulkZip(ctx, deps, path, spec)
}

func queryFraudEvidencePackFraudCH(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, clickID string, from, to time.Time) ([]FraudEvidenceFraudRowDTO, error) {
	return fraudExports.QueryFraudEvidencePackFraudCH(ctx, clickhouseQuery, campaignIDs, clickID, from, to)
}

func aggregateFraudEvidenceSignals(rows []FraudEvidenceFraudRowDTO) FraudEvidenceSignalsDTO {
	return fraudExports.AggregateFraudEvidenceSignals(rows)
}

func CalcSilentRejectRatio(silentRejectCount, eventCount int64) float64 {
	if eventCount <= 0 {
		return 0
	}
	return float64(silentRejectCount) / float64(eventCount)
}

func maskLevelFromContext(ctx context.Context) authz.MaskLevel {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return authz.MaskMasked
	}
	return snap.Mask
}

func ScrubFraudBreakdownRow(ctx context.Context, row FraudBreakdownRowDTO) FraudBreakdownRowDTO {
	snap, ok := authz.SnapshotFromContext(ctx)
	if ok && snap.Mask == authz.MaskFull {
		return row
	}
	category, label := FraudReasonToCategory(row.FraudReason)
	out := row
	out.FraudReason = ""
	out.PlacementID = ""
	out.FraudCategory = category
	out.FraudCategoryLabel = label
	return out
}

func ScrubFraudBreakdownRows(ctx context.Context, rows []FraudBreakdownRowDTO) []FraudBreakdownRowDTO {
	snap, ok := authz.SnapshotFromContext(ctx)
	if ok && snap.Mask == authz.MaskFull {
		return rows
	}
	out := make([]FraudBreakdownRowDTO, len(rows))
	for i := range rows {
		out[i] = ScrubFraudBreakdownRow(ctx, rows[i])
	}
	return out
}
