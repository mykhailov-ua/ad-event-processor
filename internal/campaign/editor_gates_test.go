package campaign

import (
	"context"
	"fmt"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCampaignGeoSummary_emptyRulesNeutralLabel(t *testing.T) {
	t.Parallel()
	out := buildCampaignGeoSummary(CampaignDTO{}, false)
	assert.Equal(t, "any country", out.IncludedLabel)
	assert.Equal(t, "none", out.ExcludedLabel)
}

func TestBuildCampaignGeoSummary_expandCapsAt50_holdout(t *testing.T) {
	t.Parallel()
	codes := make([]string, 60)
	for i := range codes {
		codes[i] = fmt.Sprintf("C%02d", i)
	}
	out := buildCampaignGeoSummary(CampaignDTO{TargetCountries: codes}, true)
	assert.True(t, out.Truncated)
	assert.Len(t, out.Expanded, campaignGeoExpandMaxRows)
}

func TestBuildCampaignEventTimeline_masksActor_holdout(t *testing.T) {
	t.Parallel()
	timeline := buildCampaignEventTimeline([]CampaignEventDTO{{
		EventType: "click",
		UserID:    "user-secret-id",
		CreatedAt: "2026-08-27T12:00:00Z",
	}}, true)
	require.Len(t, timeline.Days, 1)
	require.Len(t, timeline.Days[0].Events, 1)
	assert.Equal(t, "us***id", timeline.Days[0].Events[0].ActorLabel)
}

func TestAttachInvalidSpendKPI_noDoubleCount_holdout(t *testing.T) {
	t.Parallel()
	out := reports.BuildCustomerFraudOverview(100, 40, 10, DataFreshnessDTO{})
	reports.AttachInvalidSpendKPI(&out, 40, 10, 100, 1_000_000, 0.95)
	assert.Equal(t, int64(500_000), out.InvalidSpendMicros)
}

func TestAttachInvalidSpendKPI_lowCoverageDisclaimer(t *testing.T) {
	t.Parallel()
	out := reports.BuildCustomerFraudOverview(100, 10, 5, DataFreshnessDTO{})
	reports.AttachInvalidSpendKPI(&out, 10, 5, 100, 1_000_000, 0.5)
	assert.Contains(t, out.Disclaimer, "90%")
}

func TestAllowedActionsDirectRoute403_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{authz.PermCampaignsReadMasked: {}, authz.PermCampaignsPause: {}},
		Mask:        authz.MaskMasked,
	})
	actions, denied := computeCampaignAllowedActions(ctx, "ACTIVE")
	assert.Contains(t, actions, "pause")
	assert.NotContains(t, actions, "edit_fraud")
	assert.Equal(t, "requires_campaigns_write", denied["edit_fraud"])
}

func TestBuildCampaignFraudEditorSummary_noCorpusPaths(t *testing.T) {
	t.Parallel()
	summary := buildCampaignFraudEditorSummary(CampaignDTO{ID: "c1", CustomerID: "cust-1"}, CampaignFraudPreviewDTO{
		ByTier: FraudPreviewTierCountsDTO{Suspect: 2},
	})
	for _, card := range summary.Cards {
		assert.NotContains(t, card.ReportHref, "corpus")
		assert.NotContains(t, card.TitleLabel, "registry")
	}
}
