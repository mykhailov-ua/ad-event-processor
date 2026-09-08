package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCampaignListStatusFilter_mapsWarnings(t *testing.T) {
	status, warningsOnly := normalizeCampaignListStatusFilter("WARNINGS")
	require.Equal(t, "", status)
	require.True(t, warningsOnly)

	status, warningsOnly = normalizeCampaignListStatusFilter("ACTIVE")
	require.Equal(t, "ACTIVE", status)
	require.False(t, warningsOnly)
}

func TestCampaignDTOHasListWarning_budgetAndMargin(t *testing.T) {
	pct := 92.0
	require.True(t, CampaignDTOHasListWarning(CampaignDTO{
		Status:        "ACTIVE",
		BudgetUsedPct: &pct,
	}))
	require.True(t, CampaignDTOHasListWarning(CampaignDTO{
		Status:       "ACTIVE",
		MarginBreach: true,
	}))
	require.False(t, CampaignDTOHasListWarning(CampaignDTO{
		Status:       "PAUSED",
		MarginBreach: true,
	}))
}
