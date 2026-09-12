package entitlements

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureAllowedByKey_openrtbRequiresActiveLicense(t *testing.T) {
	ent := Entitlements{Features: FeatureSet{RtbLive: true}}
	require.False(t, FeatureAllowedByKey(StateExpired, ent, "openrtb"))
	require.True(t, FeatureAllowedByKey(StateActive, ent, "openrtb"))
}

func TestFeatureAllowedByKey_unknownKeyFailsClosed(t *testing.T) {
	ent := Entitlements{Features: FeatureSet{RtbLive: true}}
	require.False(t, FeatureAllowedByKey(StateActive, ent, "not_a_feature"))
}

func TestSanitizeFeaturesForSKU_starterBlocksMarginGuard_holdout(t *testing.T) {
	in := FeatureSet{MarginGuard: true}
	out := SanitizeFeaturesForSKU(SKUCodeStarter, in)
	require.False(t, out.MarginGuardEnabled())
}

func TestSanitizeFeaturesForSKU_proAllowsMarginGuard(t *testing.T) {
	in := FeatureSet{MarginGuard: true}
	out := SanitizeFeaturesForSKU(SKUCodePro, in)
	require.True(t, out.MarginGuardEnabled())
}

func TestSanitizeFeaturesForSKU_starterBlocksBrokerWal(t *testing.T) {
	in := FeatureSet{BrokerWal: true}
	out := SanitizeFeaturesForSKU(SKUCodeStarter, in)
	require.False(t, out.BrokerWalEnabled())
}

func TestSanitizeFeaturesForSKU_scaleAllowsBrokerWal(t *testing.T) {
	in := FeatureSet{BrokerWal: true}
	out := SanitizeFeaturesForSKU(SKUCodeScale, in)
	require.True(t, out.BrokerWalEnabled())
}
