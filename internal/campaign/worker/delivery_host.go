package worker

type CreativeMABStat struct {
	Impressions int64
	Clicks      int64
}

type AutoscaleBudgetAuditChange struct {
	OldBudget string
	NewBudget string
	CTR       float64
	Target    string
	Source    string
}

type campaignBudgetOutboxPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}
