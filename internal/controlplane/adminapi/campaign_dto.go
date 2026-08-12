package adminapi

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForecastClickHouseTimeout = errors.New("forecast clickhouse query timed out")
	ErrForecastUnavailable       = errors.New("forecast service unavailable")
	ErrClickHouseNotConfigured   = errors.New("clickhouse not configured")
)

const forecastDefaultRetryAfterSec = 30

func ForecastRetryAfterSec() int {
	return forecastDefaultRetryAfterSec
}

type CampaignDTO struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	BudgetLimit     string          `json:"budget_limit"`
	CurrentSpend    string          `json:"current_spend"`
	CustomerID      string          `json:"customer_id"`
	PacingMode      string          `json:"pacing_mode"`
	DailyBudget     string          `json:"daily_budget"`
	Timezone        string          `json:"timezone"`
	FreqLimit       int32           `json:"freq_limit"`
	FreqWindow      int32           `json:"freq_window"`
	TargetCountries []string        `json:"target_countries"`
	TargetURL       string          `json:"target_url,omitempty"`
	SafePageURL     string          `json:"safe_page_url,omitempty"`
	SafePageEnabled bool            `json:"safe_page_enabled"`
	BrandID         string          `json:"brand_id,omitempty"`
	CreativePayload json.RawMessage `json:"creative_payload,omitempty"`
	ReferrerFilter  string          `json:"referrer_filter,omitempty"`
	StartAt         string          `json:"start_at,omitempty"`
	EndAt           string          `json:"end_at,omitempty"`
	DaypartHours    []int16         `json:"daypart_hours"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type BlacklistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
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

type CampaignStatsDTO struct {
	CampaignID   string                    `json:"campaign_id"`
	CurrentSpend string                    `json:"current_spend"`
	Metrics      CampaignMetricsDTO        `json:"metrics"`
	Hourly       []CampaignHourlyBucketDTO `json:"hourly"`
	Granularity  string                    `json:"granularity"`
	From         string                    `json:"from"`
	To           string                    `json:"to"`
	Stale        bool                      `json:"stale"`
	Source       string                    `json:"source"`
	Consistency  string                    `json:"consistency"`
}

type BlacklistListResponse struct {
	Items []BlacklistDTO `json:"items"`
	Total int64          `json:"total"`
}

type MutationPreviewDTO struct {
	DryRun      bool            `json:"dry_run"`
	Action      string          `json:"action"`
	WouldChange json.RawMessage `json:"would_change"`
}

type BalanceLedgerDTO struct {
	ID              int64  `json:"id"`
	CustomerID      string `json:"customer_id"`
	CampaignID      string `json:"campaign_id,omitempty"`
	Amount          string `json:"amount"`
	Type            string `json:"type"`
	IdempotencyHash string `json:"idempotency_hash,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type CustomerBalanceDTO struct {
	CustomerID string             `json:"customer_id"`
	Balance    string             `json:"balance"`
	Currency   string             `json:"currency"`
	Ledger     []BalanceLedgerDTO `json:"ledger"`
}

type LedgerExportResult struct {
	NextCursor int64
	Truncated  bool
	Bytes      int
}

type AuditLogDTO struct {
	ID         int64           `json:"id"`
	AdminID    string          `json:"admin_id,omitempty"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id,omitempty"`
	Changes    json.RawMessage `json:"changes"`
	Metadata   json.RawMessage `json:"metadata"`
	IsMasked   bool            `json:"is_masked"`
	CreatedAt  string          `json:"created_at"`
}

type AuditLogListResponse struct {
	Items []AuditLogDTO `json:"items"`
	Total int64         `json:"total"`
}

type PatchCampaignRequest struct {
	Name             *string  `json:"name,omitempty"`
	PacingMode       *string  `json:"pacing_mode,omitempty"`
	DailyBudgetMicro *int64   `json:"daily_budget_micro,omitempty"`
	Timezone         *string  `json:"timezone,omitempty"`
	FreqLimit        *int32   `json:"freq_limit,omitempty"`
	FreqWindow       *int32   `json:"freq_window,omitempty"`
	TargetCountries  []string `json:"target_countries,omitempty"`
	TargetURL        *string  `json:"target_url,omitempty"`
	SafePageURL      *string  `json:"safe_page_url,omitempty"`
	SafePageEnabled  *bool    `json:"safe_page_enabled,omitempty"`
	ReferrerFilter   *string  `json:"referrer_filter,omitempty"`
}

type CampaignEventDTO struct {
	ClickID   string          `json:"click_id"`
	EventType string          `json:"event_type"`
	UserID    string          `json:"user_id,omitempty"`
	IP        string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt string          `json:"created_at"`
}

type CampaignEventListResponse struct {
	Items []CampaignEventDTO `json:"items"`
	Total int64              `json:"total"`
}
