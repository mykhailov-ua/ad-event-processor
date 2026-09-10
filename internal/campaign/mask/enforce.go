package mask

import (
	"ad-event-processor/internal/campaign"
)

// TouchesProtectedFields reports whether a masked-role PATCH attempts to change economics or URL fields.
func TouchesProtectedFields(req campaign.PatchCampaignRequest) bool {
	if req.BudgetLimitMicro != nil || req.BudgetLimit != nil || req.DailyBudgetMicro != nil {
		return true
	}
	if req.TargetURL != nil || req.ReferrerFilter != nil {
		return true
	}
	if req.PacingMode != nil {
		return true
	}
	return false
}
