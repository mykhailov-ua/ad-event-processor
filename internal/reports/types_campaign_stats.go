package reports

import (
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
