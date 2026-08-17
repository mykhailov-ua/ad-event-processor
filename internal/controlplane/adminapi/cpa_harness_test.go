package adminapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// CPA-M0.3 harness labels for cold-path admin work (MILESTONE.md §1.0, CPA-M3+).
const (
	// harness: campaign_patch_honest
	HarnessCampaignPatchHonest = "campaign_patch_honest"
	// harness: report_action
	HarnessReportAction = "report_action"
	// harness: buyer_dlq
	HarnessBuyerDLQ = "buyer_dlq"
	// harness: publisher_scope
	HarnessPublisherScope = "publisher_scope"
	// harness: unified_dlq
	HarnessUnifiedDLQ = "unified_dlq"
)

func TestCPAHarnessLabels_registered(t *testing.T) {
	t.Parallel()
	labels := []string{
		HarnessCampaignPatchHonest,
		HarnessReportAction,
		HarnessBuyerDLQ,
		HarnessPublisherScope,
		HarnessUnifiedDLQ,
	}
	for _, label := range labels {
		require.NotEmpty(t, label)
		require.NotContains(t, label, " ")
	}
}
