package controlplane

import (
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestScrubCampaignFields_holdoutRedactsBudget(t *testing.T) {
	t.Parallel()
	out := campaign.ScrubCampaignFields(campaign.CampaignDTO{
		BudgetLimit: "100.00",
		DailyBudget: "10.00",
		TargetURL:   "https://example.com",
	}, authz.MaskMasked)
	assert.Empty(t, out.BudgetLimit)
	assert.Equal(t, campaign.RedactedMoneyDisplay(), out.BudgetLimitDisplay)
	assert.Contains(t, out.FieldsRedacted, "budget_limit")
	assert.Empty(t, out.TargetURL)
}
