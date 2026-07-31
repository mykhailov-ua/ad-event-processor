package adminapi

import (
	"encoding/json"

	"espx/internal/licensing"
)

type SubscriptionDTO struct {
	CustomerID  string                  `json:"customer_id"`
	PlanCode    string                  `json:"plan_code"`
	Status      string                  `json:"status"`
	PeriodStart string                  `json:"period_start"`
	PeriodEnd   string                  `json:"period_end,omitempty"`
	Limits      licensing.LimitsDTO     `json:"limits"`
	Features    licensing.FeatureSetDTO `json:"features"`
	Effective   licensing.LimitsDTO     `json:"effective_limits"`
	Usage       []UsageMeterDTO         `json:"usage"`
}

type UsageMeterDTO struct {
	Meter     string `json:"meter"`
	Period    string `json:"period"`
	Value     int64  `json:"value"`
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
}

type UsageDailyDTO struct {
	CustomerID string `json:"customer_id"`
	UsageDate  string `json:"usage_date"`
	Meter      string `json:"meter"`
	Value      int64  `json:"value"`
}

type QuotaStatusDTO struct {
	CustomerID string `json:"customer_id"`
	Limit      int64  `json:"limit"`
	Value      int64  `json:"value"`
	Remaining  int64  `json:"remaining"`
	Timezone   string `json:"timezone"`
}

type UpdateSubscriptionRequest struct {
	PlanCode      string          `json:"plan_code"`
	Status        string          `json:"status"`
	PeriodStart   string          `json:"period_start"`
	PeriodEnd     string          `json:"period_end,omitempty"`
	OverridesJSON json.RawMessage `json:"overrides_json,omitempty"`
}

type QuotaBumpRequest struct {
	BonusRequests int64 `json:"bonus_requests"`
}
