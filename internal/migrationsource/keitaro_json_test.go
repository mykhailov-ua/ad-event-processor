package migrationsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrationMapsDir(t *testing.T) string {
	dir := MapsRootDir()
	if _, err := os.Stat(filepath.Join(dir, "keitaro_macros.yaml")); err != nil {
		t.Fatalf("migration maps dir missing: %s", dir)
	}
	return dir
}

func TestMigrationMaps_loadAndValidate(t *testing.T) {
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	require.NotEmpty(t, maps.KeitaroMacros)
	require.NotEmpty(t, maps.KeitaroSources)
	var facebook string
	for _, row := range maps.KeitaroSources {
		if row.KeitaroName == "Facebook" {
			facebook = row.BundledSlug
			break
		}
	}
	assert.Equal(t, "traffic_facebook", facebook)
}

func TestParseKeitaroJSON_facebookFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)
	bundle, err := ParseKeitaroJSON(raw)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	assert.Equal(t, "Facebook Camp", bundle.Campaigns[0].Name)
	assert.Equal(t, "Facebook", bundle.Campaigns[0].TrafficSourceName)
}

func TestPreviewKeitaro_facebookSub2CampaignId(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_facebook_campaign.json"))
	require.NoError(t, err)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindKeitaroJSON, raw, maps)
	require.NoError(t, err)
	require.Len(t, result.MappedCampaigns, 1)
	camp := result.MappedCampaigns[0]
	assert.Equal(t, "traffic_facebook", camp.BundledSlug)
	assert.Equal(t, "meta-facebook", camp.UITemplateID)
	assert.Equal(t, "{{campaign.id}}", camp.ClickQueryParams["sub2"])
	assert.Equal(t, "{{ad.id}}", camp.ClickQueryParams["sub4"])
	assert.Equal(t, int64(100_000_000), camp.BudgetLimitMicro)
}

func TestPreviewKeitaro_unmappedMacroWarning(t *testing.T) {
	payload := []byte(`{"campaigns":[{"id":1,"name":"Test","tracking_url":"https://trk.example/click?sub1={unknown_token}"}]}`)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindKeitaroJSON, payload, maps)
	require.NoError(t, err)
	var found bool
	for _, w := range result.Warnings {
		if w.Slug == "unmapped_macro" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected unmapped_macro warning")
}
