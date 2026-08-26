package licensing

import "strings"

const (
	SKUCodePilot      = "pilot"
	SKUCodeStarter    = "starter"
	SKUCodePro        = "pro"
	SKUCodeScale      = "scale"
	SKUCodeNetwork    = "network"
	SKUCodeEnterprise = "enterprise"
	SKUCodeLicense    = "license"
)

func SanitizeFeaturesForSKU(sku string, features FeatureSet) FeatureSet {
	out := features.Normalized()
	switch strings.ToLower(strings.TrimSpace(sku)) {
	case SKUCodePilot, SKUCodeStarter, "solo":
		out.RtbLive = false
		out.OpenRTBEngine = false
		out.EbpfXDPEdge = false
		out.IvtMLDetector = false
		out.MlFraudBoost = false
		out.MultiRegion = false
		out.SlotMigration = false
		out.ExternalResidentialIntel = false
		out.ModeratorIntelFeed = false
		out.AdPlatformCampaignAPI = false
	case SKUCodePro:
		out.RtbLive = false
		out.OpenRTBEngine = false
		out.IvtMLDetector = true
		out.MlFraudBoost = false
		out.EbpfXDPEdge = false
		out.ExternalResidentialIntel = false
		out.ModeratorIntelFeed = false
		out.AdPlatformCampaignAPI = false
	case SKUCodeScale:
		out.EbpfXDPEdge = false
	case SKUCodeNetwork, SKUCodeEnterprise, SKUCodeLicense:
	}
	return out
}

func OpenRTBAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.OpenRTBEnabled()
}

func EbpfEdgeAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.EbpfEdgeEnabled()
}

func NormalizeMaxActivationsLimit(limits Limits) int32 {
	if limits.MaxActivations == 0 {
		return 1
	}
	const maxInt32 = uint64(1<<31 - 1)
	if limits.MaxActivations > maxInt32 {
		return int32(1<<31 - 1)
	}
	return int32(limits.MaxActivations)
}
