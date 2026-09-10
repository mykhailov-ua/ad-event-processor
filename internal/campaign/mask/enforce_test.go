package mask

import (
	"testing"

	"ad-event-processor/internal/campaign"

	"github.com/stretchr/testify/assert"
)

func TestTouchesProtectedFields_budget_holdout(t *testing.T) {
	t.Parallel()
	micro := int64(100)
	assert.True(t, TouchesProtectedFields(campaign.PatchCampaignRequest{BudgetLimitMicro: &micro}))
}

func TestTouchesProtectedFields_nameOnly(t *testing.T) {
	t.Parallel()
	name := "safe rename"
	assert.False(t, TouchesProtectedFields(campaign.PatchCampaignRequest{Name: &name}))
}

func TestMaskedMutation_rejectsBudgetPatch_holdout(t *testing.T) {
	t.Parallel()
	micro := int64(1)
	if !TouchesProtectedFields(campaign.PatchCampaignRequest{BudgetLimitMicro: &micro}) {
		t.Fatal("holdout: budget patch must be protected for masked roles")
	}
}
