package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	HarnessCampaignPatchHonest = "campaign_patch_honest"

	HarnessReportAction = "report_action"

	HarnessBuyerDLQ = "buyer_dlq"

	HarnessPublisherScope = "publisher_scope"

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
