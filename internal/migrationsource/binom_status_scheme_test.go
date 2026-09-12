package migrationsource

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinomStatusScheme_mapsGoldenFixture(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/binom_status_scheme_campaign.json")
	require.NoError(t, err)
	bundle, err := ParseBinomJSON(raw)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	require.Len(t, bundle.Campaigns[0].StatusSchemeRules, 2)
	rules := bundle.Campaigns[0].StatusSchemeRules
	require.Equal(t, "hold", rules[0].WhenStatus)
	require.False(t, rules[0].FireOutbound)
	require.Equal(t, "approved", rules[1].WhenStatus)
	require.Equal(t, "sale", rules[1].SetGoalName)
}

func TestBinomStatusScheme_previewUnmappedCount_holdout(t *testing.T) {
	t.Parallel()
	rules, warnings := mapBinomStatusScheme(&binomStatusScheme{Rules: []binomStatusSchemeRule{{
		PayoutMode: "multiply",
	}}}, "binom:1")
	require.Empty(t, rules)
	require.NotEmpty(t, warnings)
}
