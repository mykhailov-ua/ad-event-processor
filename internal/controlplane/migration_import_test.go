package controlplane

import (
	"testing"

	"ad-event-processor/internal/migrationsource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportBundleFromMigrationShape_facebookMacros(t *testing.T) {
	shape := migrationsource.ExportCampaignShape{
		Name:                "FB Camp",
		BudgetLimitMicro:    50_000_000,
		TargetURL:           "https://lander.example/lp",
		TrafficTemplateID:   "meta-facebook",
		ClickQueryParams:    map[string]string{"sub2": "{{campaign.id}}"},
		IntegrationSchema:   "traffic_facebook",
		IngressCostParam:    "cost",
		PostbackURLTemplate: "https://aff.example/pb",
	}
	bundle := exportBundleFromMigrationShape(shape)
	require.Equal(t, "FB Camp", bundle.Campaign.Name)
	assert.Equal(t, "meta-facebook", bundle.Campaign.TrafficTemplateID)
	assert.Equal(t, "{{campaign.id}}", bundle.Campaign.ClickQueryParams["sub2"])
	assert.Equal(t, "traffic_facebook", bundle.IntegrationSchemaName)
	require.NotNil(t, bundle.Campaign.IngressCostConfig)
	assert.Equal(t, "cost", bundle.Campaign.IngressCostConfig.Param)
	require.NotNil(t, bundle.PostbackConfig)
}
