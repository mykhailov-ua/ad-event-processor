package entitlements

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

func TestSanitizeFeaturesForSKU_starterBlocksProUpsellFeatures_holdout(t *testing.T) {
	in := FeatureSet{
		IvtMLDetector:         true,
		MarginGuard:           true,
		AdPlatformCampaignAPI: true,
		FraudDisputeEvidence:  true,
	}
	out := SanitizeFeaturesForSKU(SKUCodeStarter, in)
	require.False(t, out.IvtMLEnabled())
	require.False(t, out.MarginGuardEnabled())
	require.False(t, out.AdPlatformCampaignAPI)
	require.False(t, out.FraudDisputeEvidenceEnabled())
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
	require.True(t, out.MarginGuardEnabled())
	require.True(t, out.AdPlatformCampaignAPI)
	require.True(t, out.FraudDisputeEvidenceEnabled())
}

func TestLoadSKUFile_proTierFeatures(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodePro)
	require.NoError(t, err)
	require.False(t, sku.Features.OpenRTBEngine)
	require.True(t, sku.Features.IvtMLDetector)
	require.False(t, sku.Features.MlFraudBoost)
	require.True(t, sku.Features.MarginGuard)
	require.True(t, sku.Features.AdPlatformCampaignAPI)
	require.True(t, sku.Features.FraudDisputeEvidence)
}

func TestSanitizeFeaturesForSKU_pilotMatchesPro_holdout(t *testing.T) {
	in := FeatureSet{
		RtbLive:       true,
		OpenRTBEngine: true,
		EbpfXDPEdge:   true,
		MlFraudBoost:  true,
		MultiRegion:   true,
		SlotMigration: true,
		IvtMLDetector: false,
		MarginGuard:   false,
	}
	out := SanitizeFeaturesForSKU(SKUCodePilot, in)
	require.False(t, out.OpenRTBEnabled())
	require.False(t, out.EbpfEdgeEnabled())
	require.False(t, out.MlFraudBoostEnabled())
	require.False(t, out.MultiRegionEnabled())
	require.False(t, out.SlotMigration)
	require.True(t, out.IvtMLEnabled())
	require.True(t, out.MarginGuard)
	require.True(t, out.AdPlatformCampaignAPI)
	require.True(t, out.FraudDisputeEvidence)
}

func TestSanitizeFeaturesForSKU_scaleAllowsExternalResidentialIntel(t *testing.T) {
	in := FeatureSet{ExternalResidentialIntel: true}
	out := SanitizeFeaturesForSKU(SKUCodeScale, in)
	require.True(t, out.ExternalResidentialIntelEnabled())

	outPro := SanitizeFeaturesForSKU(SKUCodePro, in)
	require.False(t, outPro.ExternalResidentialIntelEnabled())
}

func TestLoadSKUFile_brokerWalTierMatrix_holdout(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	scale, err := doc.GetSKU(SKUCodeScale)
	require.NoError(t, err)
	require.True(t, scale.Features.BrokerWal)
	pro, err := doc.GetSKU(SKUCodePro)
	require.NoError(t, err)
	require.False(t, pro.Features.BrokerWal)
}

func TestLoadSKUFile_costSyncNetworksUnlimited_holdout(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	for _, code := range []string{SKUCodePilot, SKUCodeStarter, SKUCodePro, SKUCodeScale} {
		sku, err := doc.GetSKU(code)
		require.NoError(t, err)
		require.Equal(t, uint64(999999), sku.Limits.MaxCostSyncNetworks, "sku=%s", code)
	}
}

func TestLoadSKUFile_fraudDisputeEvidenceTierMatrix_holdout(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	starter, err := doc.GetSKU(SKUCodeStarter)
	require.NoError(t, err)
	require.False(t, starter.Features.FraudDisputeEvidence)
	pro, err := doc.GetSKU(SKUCodePro)
	require.NoError(t, err)
	require.True(t, pro.Features.FraudDisputeEvidence)
	scale, err := doc.GetSKU(SKUCodeScale)
	require.NoError(t, err)
	require.True(t, scale.Features.FraudDisputeEvidence)
}

func TestLoadSKUFile_pilotSmokeLimits(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodePilot)
	require.NoError(t, err)
	require.Equal(t, 10, sku.ValidDays)
	require.Equal(t, uint64(0), sku.Limits.MaxRPS)
	require.Equal(t, uint64(0), sku.Limits.MaxAPIKeys)
	require.Equal(t, uint64(1), sku.Limits.MaxTenants)
	require.Equal(t, uint64(0), sku.Limits.MaxExportChunkBytes)
	require.False(t, sku.Features.RtbLive)
	require.False(t, sku.Features.OpenRTBEngine)
	require.True(t, sku.Features.MarginGuard)
	require.True(t, sku.Features.IvtMLDetector)
	require.Equal(t, uint64(999999), sku.Limits.MaxCostSyncNetworks)

	claims := sku.BuildClaims(IssueLicenseInput{
		CustomerName: "Trial",
		DeploymentID: "dep-pilot",
		LicenseID:    "lic-pilot",
	})
	sanitized := SanitizeFeaturesForSKU(claims.SKU, claims.Features)
	require.False(t, sanitized.OpenRTBEnabled())
	require.True(t, sanitized.MarginGuard)
}

func TestLoadSKUFile_noNetworkSKU_holdout(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	_, err = doc.GetSKU(SKUCodeNetwork)
	require.Error(t, err)
}

func TestOpenRTBAllowed_requiresActiveLicense(t *testing.T) {
	ent := Entitlements{Features: FeatureSet{RtbLive: true}}
	require.False(t, OpenRTBAllowed(StateExpired, ent))
	require.True(t, OpenRTBAllowed(StateActive, ent))
}
