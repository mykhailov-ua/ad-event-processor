package campaign

type IntegrationHealthStatus string

const (
	IntegrationHealthOK   IntegrationHealthStatus = "ok"
	IntegrationHealthWarn IntegrationHealthStatus = "warn"
	IntegrationHealthFail IntegrationHealthStatus = "fail"
)

type IntegrationHealthRow struct {
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	FixRoute     string `json:"fix_route,omitempty"`
	ActionID     string `json:"action_id,omitempty"`
	FixHintLabel string `json:"fix_hint_label,omitempty"`
}

type IntegrationHealthDTO struct {
	CampaignID string                 `json:"campaign_id"`
	Summary    string                 `json:"summary"`
	Rows       []IntegrationHealthRow `json:"rows"`
}
