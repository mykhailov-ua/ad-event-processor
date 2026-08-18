package controlplane

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

type OffsetListResponse[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type CursorListResponse[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total"`
	NextCursor string `json:"next_cursor"`
	Limit      int32  `json:"limit"`
}

type ItemsResponse[T any] struct {
	Items []T `json:"items"`
}

type CampaignMarginDTO struct {
	CampaignID           string `json:"campaign_id"`
	WindowStart          string `json:"window_start"`
	WindowHours          int    `json:"window_hours"`
	AdvertiserSpendMicro int64  `json:"advertiser_spend_micro"`
	RtbCostMicro         int64  `json:"rtb_cost_micro"`
	OperatorMarginMicro  int64  `json:"operator_margin_micro"`
	PublisherPayoutMicro int64  `json:"publisher_payout_micro"`
	CostOverRevenueLimit int64  `json:"cost_over_revenue_limit"`
	ThresholdBps         int    `json:"threshold_bps"`
	MarginBreach         bool   `json:"margin_breach"`
}

type MarginGuardActivityRow struct {
	ID          uuid.UUID       `json:"id"`
	PolicyID    uuid.UUID       `json:"policy_id"`
	CampaignID  uuid.UUID       `json:"campaign_id"`
	PlacementID string          `json:"placement_id"`
	Action      string          `json:"action"`
	Reason      string          `json:"reason"`
	Metrics     json.RawMessage `json:"metrics"`
	CreatedAt   time.Time       `json:"created_at"`
}

type PaymentIntentListResponse = OffsetListResponse[PaymentHistoryRow]

type PaymentIntentCreatedResponse struct {
	IntentID       string `json:"intent_id"`
	Status         string `json:"status"`
	CheckoutURL    string `json:"checkout_url"`
	ProviderRef    string `json:"provider_ref"`
	DepositAddress string `json:"deposit_address,omitempty"`
	DepositNetwork string `json:"deposit_network,omitempty"`
	DepositQRSVG   string `json:"deposit_qr_svg,omitempty"`
}

type ExportJobCreatedResponse struct {
	JobID string `json:"job_id"`
}

type IDCreatedResponse struct {
	ID string `json:"id"`
}

type APIKeyCreatedResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RawKey    string `json:"raw_key"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type DeliveryListResponse = ItemsResponse[DeliveryDTO]

type LedgerLinesListResponse = CursorListResponse[LedgerLineDTO]
