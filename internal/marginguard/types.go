package marginguard

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ActivityRow struct {
	ID          uuid.UUID       `json:"id"`
	PolicyID    uuid.UUID       `json:"policy_id"`
	CampaignID  uuid.UUID       `json:"campaign_id"`
	PlacementID string          `json:"placement_id"`
	Action      string          `json:"action"`
	Reason      string          `json:"reason"`
	Metrics     json.RawMessage `json:"metrics"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CampaignMargin struct {
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

type pausePlacementOutboxPayload struct {
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id"`
	Action      string `json:"action,omitempty"`
}
