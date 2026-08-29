package integration

import (
	"testing"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCampaignIntegrationHealth_metaMissingCredentialWarns(t *testing.T) {
	campaignID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	out := BuildCampaignIntegrationHealth(IntegrationHealthInput{
		CampaignID:        campaignID,
		TrafficTemplateID: "meta-facebook",
		TargetURL:         "https://lander.example/",
		ClickQueryParams: map[string]string{
			"ad_campaign_id": "{{campaign.id}}",
			"sub2":           "{{campaign.id}}",
			"sub4":           "{{ad.id}}",
			"fbclid":         "{{fbclid}}",
		},
		CostSyncNetwork:           "facebook",
		CostSyncCredentialPresent: false,
	})
	require.Equal(t, "warn", out.Summary)
	var credRow *campaign.IntegrationHealthRow
	for i := range out.Rows {
		if out.Rows[i].Slug == "cost_sync_credential" {
			credRow = &out.Rows[i]
			break
		}
	}
	require.NotNil(t, credRow)
	assert.Equal(t, "warn", credRow.Status)
	assert.Contains(t, credRow.Message, "facebook")
}

func TestBuildCampaignIntegrationHealth_missingJoinKeysFail(t *testing.T) {
	campaignID := uuid.New()
	out := BuildCampaignIntegrationHealth(IntegrationHealthInput{
		CampaignID:        campaignID,
		TrafficTemplateID: "meta-facebook",
		TargetURL:         "https://lander.example/",
		ClickQueryParams:  map[string]string{"sub1": "{subid}"},
		CostSyncNetwork:   "facebook",
	})
	assert.Equal(t, "fail", out.Summary)
}
