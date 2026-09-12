package controlplane

import (
	"testing"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"

	"github.com/stretchr/testify/require"
)

func TestMarginGuardLicense_starterSanitizeBlocksFeature_holdout(t *testing.T) {
	features := entitlements.SanitizeFeaturesForSKU(entitlements.SKUCodeStarter, entitlements.FeatureSet{
		MarginGuard: true,
	})
	ent := licensing.Entitlements{Features: features}
	allowed := licensing.FeatureAllowedByKey(licensing.StateActive, ent, "margin_guard")
	require.False(t, allowed)
}

func TestMarginGuardLicense_proAllowsFeature(t *testing.T) {
	features := entitlements.SanitizeFeaturesForSKU(entitlements.SKUCodePro, entitlements.FeatureSet{
		MarginGuard: true,
	})
	ent := licensing.Entitlements{Features: features}
	allowed := licensing.FeatureAllowedByKey(licensing.StateActive, ent, "margin_guard")
	require.True(t, allowed)
}
