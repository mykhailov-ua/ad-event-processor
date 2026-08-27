package migrationsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeitaroJSON_adminAPIWire_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_admin_api_campaigns.json"))
	require.NoError(t, err)
	_, err = ParseKeitaroJSON(raw)
	require.Error(t, err)
	bundle, err := ParseKeitaroAdminAPI(raw)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.Campaigns)
}

func TestParseKeitaroAdminAPI_domainAlias_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_admin_api_campaigns.json"))
	require.NoError(t, err)
	bundle, err := ParseKeitaroAdminAPI(raw)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	camp := bundle.Campaigns[0]
	assert.Equal(t, "Facebook Camp", camp.Name)
	assert.Equal(t, "Facebook", camp.TrafficSourceName)
	assert.Contains(t, camp.TrackingURL, "/fb-camp?")
	assert.Contains(t, camp.TrackingURL, "sub1={subid}")
	assert.Equal(t, float64(0), camp.BudgetUSD)
}

func TestPreviewKeitaroAdminAPI_costValueNotBudget_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_admin_api_campaigns.json"))
	require.NoError(t, err)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindKeitaroAdminAPI, raw, maps)
	require.NoError(t, err)
	require.Len(t, result.MappedCampaigns, 1)
	camp := result.MappedCampaigns[0]
	assert.Equal(t, int64(0), camp.BudgetLimitMicro)
	assert.Equal(t, "traffic_facebook", camp.BundledSlug)
	assert.Equal(t, "{{campaign.id}}", camp.ClickQueryParams["sub2"])
}

func TestParseBinomJSON_reportAPIWire_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "binom_campaign_report.json"))
	require.NoError(t, err)
	_, err = ParseBinomJSON(raw)
	require.Error(t, err)
	bundle, err := ParseBinomReportAPI(raw)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.Campaigns)
}

func TestParseBinomReportAPI_spendNotBudget_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "binom_campaign_report.json"))
	require.NoError(t, err)
	bundle, err := ParseBinomReportAPI(raw)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	assert.Equal(t, "binom:7", bundle.Campaigns[0].Ref)
	assert.Equal(t, float64(0), bundle.Campaigns[0].BudgetUSD)
}

func TestPreviewBinomReportAPI_sub2CampaignId_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "binom_campaign_report.json"))
	require.NoError(t, err)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindBinomReportAPI, raw, maps)
	require.NoError(t, err)
	require.Len(t, result.MappedCampaigns, 1)
	camp := result.MappedCampaigns[0]
	assert.Equal(t, int64(0), camp.BudgetLimitMicro)
	assert.Equal(t, "traffic_facebook", camp.BundledSlug)
	assert.Equal(t, "{{campaign.id}}", camp.ClickQueryParams["sub2"])
}
