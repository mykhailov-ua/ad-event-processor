package reports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForecastClickHouseTimeout = errors.New("forecast clickhouse query timed out")
	ErrForecastUnavailable       = errors.New("forecast service unavailable")
	ErrClickHouseNotConfigured   = errors.New("clickhouse not configured")
	ErrInvalidTimeRange          = errors.New("invalid time range")
)

const forecastDefaultRetryAfterSec = 30

func ForecastRetryAfterSec() int {
	return forecastDefaultRetryAfterSec
}

type CampaignForecastInput struct {
	CustomerID       *uuid.UUID
	BudgetLimitMicro int64
	TargetCountries  []string
	DaypartHours     []int16
	StartAt          time.Time
	EndAt            time.Time
	PacingMode       string
	Timezone         string
}

type SpendCurvePoint struct {
	Hour        string `json:"hour"`
	SpendMicro  int64  `json:"spend_micro"`
	Impressions int64  `json:"impressions"`
}

type ForecastAdvisory struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	SuggestedPacing string `json:"suggested_pacing"`
}

type CampaignForecastDTO struct {
	ImpressionsP50 int64             `json:"impressions_p50"`
	ImpressionsP90 int64             `json:"impressions_p90"`
	SpendCurve     []SpendCurvePoint `json:"spend_curve"`
	LowConfidence  bool              `json:"low_confidence"`
	Advisory       *ForecastAdvisory `json:"advisory,omitempty"`
}

type CampaignMetricsDTO struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	Conversions int64 `json:"conversions"`
}

type CampaignHourlyBucketDTO struct {
	Hour        string `json:"hour"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignDailyBucketDTO struct {
	Day         string `json:"day"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignStatsDTO struct {
	CampaignID   string                    `json:"campaign_id"`
	CurrentSpend string                    `json:"current_spend"`
	Metrics      CampaignMetricsDTO        `json:"metrics"`
	Hourly       []CampaignHourlyBucketDTO `json:"hourly"`
	Daily        []CampaignDailyBucketDTO  `json:"daily,omitempty"`
	Granularity  string                    `json:"granularity"`
	From         string                    `json:"from"`
	To           string                    `json:"to"`
	Stale        bool                      `json:"stale"`
	Source       string                    `json:"source"`
	Consistency  string                    `json:"consistency"`
}

type auditCampaignFraudChange struct {
	FraudThresholdPass       uint8 `json:"fraud_threshold_pass"`
	FraudThresholdSuspect    uint8 `json:"fraud_threshold_suspect"`
	FraudThresholdIVT        uint8 `json:"fraud_threshold_ivt"`
	FraudThresholdBlock      uint8 `json:"fraud_threshold_block"`
	SilentRejectEnabled      bool  `json:"silent_reject_enabled"`
	BehaviorFlags            int32 `json:"behavior_flags"`
	CanvasRetestEnabled      bool  `json:"canvas_retest_enabled"`
	CgnatIPPolicyEnabled     bool  `json:"cgnat_ip_policy_enabled"`
	AcceptLangGeoEnabled     bool  `json:"accept_lang_geo_enabled"`
	JSONSerializationEnabled bool  `json:"json_serialization_enabled"`
}

type SourceRowDTO struct {
	CampaignID   string  `json:"campaign_id"`
	Sub1         string  `json:"sub1,omitempty"`
	Sub2         string  `json:"sub2,omitempty"`
	Country      string  `json:"country,omitempty"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	CPAMicro     int64   `json:"cpa_micro"`
	ROIPct       float64 `json:"roi_pct"`
	CTR          float64 `json:"ctr"`
	IVTRate      float64 `json:"ivt_rate"`
	QualityScore float64 `json:"quality_score"`
}

type FraudGeoHintDTO struct {
	Country    string  `json:"country"`
	IVTRate    float64 `json:"ivt_rate"`
	IVTEvents  int64   `json:"ivt_events"`
	Clicks     int64   `json:"clicks"`
	CampaignID string  `json:"campaign_id,omitempty"`
}

type EdgeMetricsPanelDTO struct {
	UpdatedAt      string            `json:"updated_at,omitempty"`
	IngressH1      uint64            `json:"ingress_h1"`
	IngressH2      uint64            `json:"ingress_h2"`
	IngressH3      uint64            `json:"ingress_h3"`
	BodyStream     uint64            `json:"body_stream"`
	BodyPeek       uint64            `json:"body_peek"`
	BodyRead       uint64            `json:"body_read"`
	Blocked        map[string]uint64 `json:"blocked"`
	TarpitTotal    uint64            `json:"tarpit_total"`
	BlacklistStale uint64            `json:"blacklist_stale"`
}

type DataFreshnessDTO struct {
	AsOf         string                   `json:"as_of"`
	Consistency  string                   `json:"consistency"`
	Stale        bool                     `json:"stale"`
	CHLagSeconds int                      `json:"ch_lag_seconds,omitempty"`
	Sources      []DataSourceFreshnessDTO `json:"sources,omitempty"`
}

type BuyerAttentionDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type BuyerCampaignPortfolioRowDTO struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Status                  string  `json:"status"`
	PacingMode              string  `json:"pacing_mode"`
	Impressions7d           int64   `json:"impressions_7d"`
	Clicks7d                int64   `json:"clicks_7d"`
	SpendMicro              int64   `json:"spend_micro,omitempty"`
	BudgetMicro             int64   `json:"budget_micro,omitempty"`
	UtilizationPct          float64 `json:"utilization_pct,omitempty"`
	PacingDriftPct          float64 `json:"pacing_drift_pct,omitempty"`
	EstimatedPacingDriftPct float64 `json:"estimated_pacing_drift_pct,omitempty"`
	OverspendRisk           bool    `json:"overspend_risk,omitempty"`
	MarginBreach            bool    `json:"margin_breach,omitempty"`
}

type BuyerPortfolioDTO struct {
	CustomerID     string                         `json:"customer_id"`
	Active         int                            `json:"active"`
	Paused         int                            `json:"paused"`
	Archived       int                            `json:"archived"`
	Impressions7d  int64                          `json:"impressions_7d"`
	Clicks7d       int64                          `json:"clicks_7d"`
	OverspendCount int                            `json:"overspend_count,omitempty"`
	Attention      []BuyerAttentionDTO            `json:"attention"`
	Campaigns      []BuyerCampaignPortfolioRowDTO `json:"campaigns"`
	Fraud          *CustomerFraudOverviewDTO      `json:"fraud,omitempty"`
}

type BuyerPortfolioReader interface {
	GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error)
	GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (BuyerPortfolioDTO, error)
}

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

type FraudBreakdownReportResponse struct {
	Rows       []FraudBreakdownRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
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

type IVTBySourceReportResponse struct {
	Rows       []IVTBySourceRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO    `json:"freshness"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type IVTBySourceCHRow struct {
	CampaignID  string
	Sub1        string
	Sub2        string
	Country     string
	Impressions int64
	Clicks      int64
	IVTEvents   int64
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

type SilentRejectImpressionFunnelReportResponse struct {
	Rows       []SilentRejectImpressionFunnelRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO                     `json:"freshness"`
	NextCursor string                               `json:"next_cursor,omitempty"`
}

type MLShadowDeltaSnapshot struct {
	RangeFrom   time.Time
	RangeTo     time.Time
	GeneratedAt time.Time
	Rows        []map[string]any
}

type MLScoreBucketDTO struct {
	ScoreBucket float64 `json:"score_bucket"`
	RowCount    int64   `json:"row_count"`
}

type MLShadowDeltaRowDTO struct {
	Bucket           string  `json:"bucket"`
	AvgShadowScore   float64 `json:"avg_shadow_score"`
	ScoreCount       int64   `json:"score_count"`
	AvgFeatureEvents float64 `json:"avg_feature_events"`
}

type MLFeatureSpikeRowDTO struct {
	WindowStart string `json:"window_start"`
	Events      int64  `json:"events"`
	Clicks      int64  `json:"clicks"`
	Campaigns   int64  `json:"campaigns"`
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

type CustomerFraudByDimensionReportResponse struct {
	Rows       []CustomerFraudByDimensionRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO                 `json:"freshness"`
	Truncated  bool                             `json:"truncated,omitempty"`
	NextCursor string                           `json:"next_cursor,omitempty"`
}

type DimensionAggKey struct {
	CampaignID string
	Value      string
}

type DimensionTopCategorySlot struct {
	Category string
	Events   int64
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

type CustomerFraudByTypeReportResponse struct {
	Rows       []CustomerFraudByTypeRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO            `json:"freshness"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}

type FraudCategoryAggKey struct {
	CampaignID string
	Category   string
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

type WireSignalBreakdownRowDTO struct {
	CampaignID         string  `json:"campaign_id"`
	FraudReason        string  `json:"fraud_reason,omitempty"`
	FraudCategory      string  `json:"fraud_category,omitempty"`
	FraudCategoryLabel string  `json:"fraud_category_label,omitempty"`
	EventCount         int64   `json:"event_count"`
	SilentRejectCount  int64   `json:"silent_reject_count"`
	SilentRejectRatio  float64 `json:"silent_reject_ratio"`
	SignalsDegraded    bool    `json:"signals_degraded,omitempty"`
}

type WireSignalBreakdownReportResponse struct {
	Rows       []WireSignalBreakdownRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO            `json:"freshness"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}

type SignalEffectivenessRowDTO struct {
	SignalCode          string  `json:"signal_code"`
	FraudCategory       string  `json:"fraud_category,omitempty"`
	FraudCategoryLabel  string  `json:"fraud_category_label,omitempty"`
	EventVolume         int64   `json:"event_volume"`
	BlockRate           float64 `json:"block_rate"`
	BlockRateDisplay    string  `json:"block_rate_display"`
	SilentRejectRate    float64 `json:"silent_reject_rate"`
	SilentRejectDisplay string  `json:"silent_reject_rate_display"`
	SuggestedWeightTier string  `json:"suggested_weight_tier"`
}

type SignalEffectivenessReportResponse struct {
	Rows       []SignalEffectivenessRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO            `json:"freshness"`
	NextCursor string                      `json:"next_cursor,omitempty"`
}

type LayerDesyncDrilldownRowDTO struct {
	FraudReason        string `json:"fraud_reason,omitempty"`
	FraudCategory      string `json:"fraud_category,omitempty"`
	FraudCategoryLabel string `json:"fraud_category_label,omitempty"`
	EventCount         int64  `json:"event_count"`
	SilentRejectCount  int64  `json:"silent_reject_count"`
	SignalsDegraded    bool   `json:"signals_degraded,omitempty"`
}

type LayerDesyncDrilldownSeriesPointDTO struct {
	Label             string `json:"label"`
	EventCount        int64  `json:"event_count"`
	SilentRejectCount int64  `json:"silent_reject_count"`
}

type LayerDesyncDrilldownReportResponse struct {
	Rows       []LayerDesyncDrilldownRowDTO         `json:"rows"`
	Series     []LayerDesyncDrilldownSeriesPointDTO `json:"series,omitempty"`
	Freshness  DataFreshnessDTO                     `json:"freshness"`
	NextCursor string                               `json:"next_cursor,omitempty"`
}

type LayerDesyncSummaryRowDTO struct {
	CampaignID        string `json:"campaign_id"`
	LayerDesyncCount  uint8  `json:"layer_desync_count"`
	EventCount        int64  `json:"event_count"`
	SilentRejectCount int64  `json:"silent_reject_count"`
}

type LayerDesyncSummaryReportResponse struct {
	Rows       []LayerDesyncSummaryRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO           `json:"freshness"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}
