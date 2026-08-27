package migrationsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBinomJSON_facebookFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "binom_facebook_campaign.json"))
	require.NoError(t, err)
	bundle, err := ParseBinomJSON(raw)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	assert.Equal(t, "Binom Facebook Camp", bundle.Campaigns[0].Name)
	assert.Equal(t, "Facebook", bundle.Campaigns[0].TrafficSourceName)
}

func TestPreviewBinom_facebookSub2CampaignId(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "binom_facebook_campaign.json"))
	require.NoError(t, err)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindBinomJSON, raw, maps)
	require.NoError(t, err)
	require.Len(t, result.MappedCampaigns, 1)
	camp := result.MappedCampaigns[0]
	assert.Equal(t, "traffic_facebook", camp.BundledSlug)
	assert.Equal(t, "meta-facebook", camp.UITemplateID)
	assert.Equal(t, "{{campaign.id}}", camp.ClickQueryParams["sub2"])
}
