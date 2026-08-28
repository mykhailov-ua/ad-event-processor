package platformsync

import "strings"

const (
	NetworkFacebook     = "facebook"
	NetworkGoogle       = "google"
	NetworkTikTok       = "tiktok"
	NetworkMicrosoftAds = "microsoft_ads"
)

const (
	ActionPause          = "pause"
	ActionResume         = "resume"
	ActionSetDailyBudget = "set_daily_budget"
)

const (
	MutationPending = "pending"
	MutationApplied = "applied"
	MutationFailed  = "failed"
)

var supportedNetworks = map[string]struct{}{
	NetworkFacebook:     {},
	NetworkGoogle:       {},
	NetworkTikTok:       {},
	NetworkMicrosoftAds: {},
}

func NormalizeNetwork(network string) string {
	return strings.ToLower(strings.TrimSpace(network))
}

func NetworkSupported(network string) bool {
	_, ok := supportedNetworks[NormalizeNetwork(network)]
	return ok
}

type RemoteCampaignStatus struct {
	Status              string
	DailyBudgetMicro    int64
	BudgetResource      string
	HasDailyBudgetMicro bool
}

type MutationRequest struct {
	DailyBudgetMicro int64 `json:"daily_budget_micro,omitempty"`
}

type MutationPreview struct {
	DryRun      bool   `json:"dry_run"`
	Action      string `json:"action"`
	Network     string `json:"network"`
	CampaignID  string `json:"campaign_id"`
	StatusFrom  string `json:"status_from,omitempty"`
	StatusTo    string `json:"status_to,omitempty"`
	BudgetFrom  int64  `json:"budget_from_micro,omitempty"`
	BudgetTo    int64  `json:"budget_to_micro,omitempty"`
	Noop        bool   `json:"noop,omitempty"`
	VendorWrite bool   `json:"vendor_write"`
}
