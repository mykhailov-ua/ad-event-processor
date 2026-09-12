package reports

import (
	"testing"

	"ad-event-processor/internal/automation"

	"github.com/stretchr/testify/require"
)

func TestReportRule_buildPauseCampaignFromSnapshot_holdout(t *testing.T) {
	req, meta, err := BuildAutomationRuleFromReport(CreateReportRuleRequest{
		CustomerID: "00000000-0000-4000-8000-000000000001",
		CampaignID: "00000000-0000-4000-8000-000000000010",
		ReportKey:  "spend-velocity",
		Action:     automation.ActionPauseCampaign,
		FilterSnapshot: map[string]string{
			"from": "2026-09-01",
			"to":   "2026-09-12",
		},
	})
	require.NoError(t, err)
	require.Equal(t, automation.GroupByCampaign, req.GroupBy)
	require.Len(t, req.Actions, 1)
	require.Equal(t, automation.ActionPauseCampaign, req.Actions[0].Type)
	require.Equal(t, "spend_micro", req.Metric)
	require.NotEmpty(t, meta["filter_snapshot"])
}

func TestReportRule_buildBlacklistPlacement_holdout(t *testing.T) {
	req, _, err := BuildAutomationRuleFromReport(CreateReportRuleRequest{
		CustomerID: "00000000-0000-4000-8000-000000000001",
		CampaignID: "00000000-0000-4000-8000-000000000010",
		ReportKey:  "fraud-breakdown",
		Action:     automation.ActionBlacklistPlacement,
		Threshold:  30,
		FilterSnapshot: map[string]string{
			"placement_id": "slot-1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, automation.GroupByPlacement, req.GroupBy)
	require.Equal(t, float64(30), req.Threshold)
}

func TestReportRule_requiresFilterSnapshot_holdout(t *testing.T) {
	_, _, err := BuildAutomationRuleFromReport(CreateReportRuleRequest{
		CustomerID: "00000000-0000-4000-8000-000000000001",
		CampaignID: "00000000-0000-4000-8000-000000000010",
		ReportKey:  "fraud-breakdown",
		Action:     automation.ActionBlacklistPlacement,
	})
	require.Error(t, err)
}
