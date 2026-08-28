package campaign

type CampaignTemplateDTO struct {
	ID              string   `json:"id"`
	CustomerID      string   `json:"customer_id"`
	Name            string   `json:"name"`
	BudgetLimit     string   `json:"budget_limit"`
	PacingMode      string   `json:"pacing_mode"`
	DailyBudget     string   `json:"daily_budget"`
	Timezone        string   `json:"timezone"`
	FreqLimit       int32    `json:"freq_limit"`
	FreqWindow      int32    `json:"freq_window"`
	TargetCountries []string `json:"target_countries"`
	BrandID         string   `json:"brand_id,omitempty"`
	DaypartHours    []int16  `json:"daypart_hours"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type CampaignTemplateListResponse struct {
	Items []CampaignTemplateDTO `json:"items"`
	Total int64                 `json:"total"`
}
