package integrationschema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGMM4_BundledTemplatesParse(t *testing.T) {
	for _, entry := range GMM4TemplateCatalog {
		entry := entry
		t.Run(entry.Name, func(t *testing.T) {
			t.Parallel()
			_, kind, parsed, err := LoadBundledTemplate(entry)
			require.NoError(t, err, entry.File)
			require.NotNil(t, parsed)
			require.Equal(t, entry.Kind, kind)
		})
	}
}

func TestBuildInboundTrackingURL_PropellerAds(t *testing.T) {
	entry, ok := FindCatalogEntry("traffic_propellerads")
	require.True(t, ok)
	_, _, parsed, err := LoadBundledTemplate(entry)
	require.NoError(t, err)
	inbound := parsed.(*InboundTokensSchema)
	tpl := BuildInboundTrackingURL("trk.example.com", inbound)
	require.Contains(t, tpl, "https://trk.example.com/click?")
	require.Contains(t, tpl, "campaign_id={campaign_id}")
	require.Contains(t, tpl, "sub1={sub1}")
	require.Contains(t, tpl, "zone_id={zone_id}")
}
