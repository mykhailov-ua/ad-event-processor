package licensing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeFeaturesForSKU_soloBlocksOpenRTBAndXDP(t *testing.T) {
	in := FeatureSet{
		RtbLive:       true,
		OpenRTBEngine: true,
		EbpfXDPEdge:   true,
		MlFraudBoost:  true,
	}
	out := SanitizeFeaturesForSKU(SKUCodeSolo, in)
	require.False(t, out.OpenRTBEnabled())
	require.False(t, out.EbpfEdgeEnabled())
	require.False(t, out.MlFraudBoostEnabled())
}

func TestSanitizeFeaturesForSKU_proAllowsRTBBlocksXDP(t *testing.T) {
	in := FeatureSet{RtbLive: true, EbpfXDPEdge: true}
	out := SanitizeFeaturesForSKU(SKUCodePro, in)
	require.True(t, out.OpenRTBEnabled())
	require.False(t, out.EbpfEdgeEnabled())
}

func TestOpenRTBAllowed_requiresActiveLicense(t *testing.T) {
	ent := Entitlements{Features: FeatureSet{RtbLive: true}}
	require.False(t, OpenRTBAllowed(StateExpired, ent))
	require.True(t, OpenRTBAllowed(StateActive, ent))
}
