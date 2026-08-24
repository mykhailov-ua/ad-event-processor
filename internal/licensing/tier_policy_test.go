package licensing

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeFeaturesForSKU_starterBlocksOpenRTBAndXDP(t *testing.T) {
	in := FeatureSet{
		RtbLive:       true,
		OpenRTBEngine: true,
		EbpfXDPEdge:   true,
		MlFraudBoost:  true,
	}
	out := SanitizeFeaturesForSKU(SKUCodeStarter, in)
	require.False(t, out.OpenRTBEnabled())
	require.False(t, out.EbpfEdgeEnabled())
	require.False(t, out.MlFraudBoostEnabled())
}

func TestSanitizeFeaturesForSKU_proAllowsIVTBlocksOpenRTBAndXDP(t *testing.T) {
	in := FeatureSet{
		RtbLive:       true,
		OpenRTBEngine: true,
		EbpfXDPEdge:   true,
		IvtMLDetector: false,
		MlFraudBoost:  true,
	}
	out := SanitizeFeaturesForSKU(SKUCodePro, in)
	require.False(t, out.OpenRTBEnabled())
	require.True(t, out.IvtMLEnabled())
	require.False(t, out.MlFraudBoostEnabled())
	require.False(t, out.EbpfEdgeEnabled())
}

func TestLoadSKUFile_proTierFeatures(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodePro)
	require.NoError(t, err)
	require.False(t, sku.Features.OpenRTBEngine)
	require.True(t, sku.Features.IvtMLDetector)
	require.False(t, sku.Features.MlFraudBoost)
}

func TestSanitizeFeaturesForSKU_pilotBlocksOpenRTB(t *testing.T) {
	in := FeatureSet{
		RtbLive:       true,
		OpenRTBEngine: true,
		EbpfXDPEdge:   true,
		MlFraudBoost:  true,
		MultiRegion:   true,
		SlotMigration: true,
		IvtMLDetector: true,
		MarginGuard:   true,
	}
	out := SanitizeFeaturesForSKU(SKUCodePilot, in)
	require.False(t, out.OpenRTBEnabled())
	require.False(t, out.EbpfEdgeEnabled())
	require.False(t, out.MlFraudBoostEnabled())
	require.False(t, out.MultiRegionEnabled())
	require.False(t, out.SlotMigration)
	require.False(t, out.IvtMLEnabled())
	require.True(t, out.MarginGuard)
}

func TestSanitizeFeaturesForSKU_scaleAllowsExternalResidentialIntel(t *testing.T) {
	in := FeatureSet{ExternalResidentialIntel: true}
	out := SanitizeFeaturesForSKU(SKUCodeScale, in)
	require.True(t, out.ExternalResidentialIntelEnabled())

	outPro := SanitizeFeaturesForSKU(SKUCodePro, in)
	require.False(t, outPro.ExternalResidentialIntelEnabled())
}

func TestLoadSKUFile_pilotSmokeLimits(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodePilot)
	require.NoError(t, err)
	require.Equal(t, 14, sku.ValidDays)
	require.Equal(t, uint64(5000), sku.Limits.MaxRPS)
	require.Equal(t, uint64(3), sku.Limits.MaxAPIKeys)
	require.Equal(t, uint64(1), sku.Limits.MaxTenants)
	require.Equal(t, uint64(0), sku.Limits.MaxExportChunkBytes)
	require.False(t, sku.Features.RtbLive)
	require.False(t, sku.Features.OpenRTBEngine)
	require.True(t, sku.Features.MarginGuard)

	claims := sku.BuildClaims(IssueLicenseInput{
		CustomerName: "Trial",
		DeploymentID: "dep-pilot",
		LicenseID:    "lic-pilot",
	})
	sanitized := SanitizeFeaturesForSKU(claims.SKU, claims.Features)
	require.False(t, sanitized.OpenRTBEnabled())
}

func TestOpenRTBAllowed_requiresActiveLicense(t *testing.T) {
	ent := Entitlements{Features: FeatureSet{RtbLive: true}}
	require.False(t, OpenRTBAllowed(StateExpired, ent))
	require.True(t, OpenRTBAllowed(StateActive, ent))
}
