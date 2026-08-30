package trafficoptimizer

import "time"

type RuleDTO struct {
	ID                  string     `json:"id"`
	CustomerID          string     `json:"customer_id"`
	CampaignID          string     `json:"campaign_id,omitempty"`
	FlowID              string     `json:"flow_id,omitempty"`
	BrandID             string     `json:"brand_id,omitempty"`
	Name                string     `json:"name"`
	Scope               string     `json:"scope"`
	Objective           string     `json:"objective"`
	Algorithm           string     `json:"algorithm"`
	LookbackMinutes     int        `json:"lookback_minutes"`
	MinClicks           int        `json:"min_clicks"`
	MinConversions      int        `json:"min_conversions"`
	MinSpendMicro       int64      `json:"min_spend_micro"`
	EvalIntervalMinutes int        `json:"eval_interval_minutes"`
	CooldownMinutes     int        `json:"cooldown_minutes"`
	MaxWeightDeltaPct   int        `json:"max_weight_delta_pct"`
	PresetKey           string     `json:"preset_key,omitempty"`
	Enabled             bool       `json:"enabled"`
	LastEvaluatedAt     *time.Time `json:"last_evaluated_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type UpsertRuleRequest struct {
	CustomerID          string             `json:"customer_id"`
	CampaignID          string             `json:"campaign_id,omitempty"`
	FlowID              string             `json:"flow_id,omitempty"`
	BrandID             string             `json:"brand_id,omitempty"`
	Name                string             `json:"name"`
	Scope               string             `json:"scope"`
	Objective           string             `json:"objective"`
	Algorithm           string             `json:"algorithm"`
	LookbackMinutes     int                `json:"lookback_minutes"`
	MinClicks           int                `json:"min_clicks"`
	MinConversions      int                `json:"min_conversions"`
	MinSpendMicro       int64              `json:"min_spend_micro"`
	EvalIntervalMinutes int                `json:"eval_interval_minutes"`
	CooldownMinutes     int                `json:"cooldown_minutes"`
	MaxWeightDeltaPct   int                `json:"max_weight_delta_pct"`
	PresetKey           string             `json:"preset_key"`
	PresetParameters    map[string]float64 `json:"preset_parameters"`
	Enabled             bool               `json:"enabled"`
}

type DryRunResponse struct {
	StaleWeights bool              `json:"stale_weights"`
	Arms         []DryRunArmResult `json:"arms"`
}

type DryRunArmResult struct {
	EntityID       string  `json:"entity_id"`
	CurrentWeight  int32   `json:"current_weight"`
	ProposedWeight int32   `json:"proposed_weight"`
	ObservedValue  float64 `json:"observed_value"`
}
