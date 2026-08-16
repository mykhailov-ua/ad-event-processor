package licensing

import "strings"

// Commercial SKU codes (deploy/vendor/sku.yaml). Used to sanitize signed JWT features at runtime.
const (
	SKUCodePilot      = "pilot"
	SKUCodeStarter    = "starter"
	SKUCodePro        = "pro"
	SKUCodeScale      = "scale"
	SKUCodeNetwork    = "network"
	SKUCodeEnterprise = "enterprise"
	SKUCodeLicense    = "license" // internal unlimited
)

// SanitizeFeaturesForSKU forces tier feature caps (HTTP redirect + track only on entry tiers).
func SanitizeFeaturesForSKU(sku string, features FeatureSet) FeatureSet {
	out := features.Normalized()
	switch strings.ToLower(strings.TrimSpace(sku)) {
	case SKUCodeStarter, "solo":
		out.RtbLive = false
		out.OpenRTBEngine = false
		out.EbpfXDPEdge = false
		out.IvtMLDetector = false
		out.MlFraudBoost = false
		out.MultiRegion = false
		out.SlotMigration = false
	case SKUCodePro, SKUCodeScale:
		out.EbpfXDPEdge = false
	case SKUCodeNetwork:
		// RTB + ML; XDP still Enterprise-only unless explicitly granted in JWT.
		out.EbpfXDPEdge = false
	case SKUCodePilot:
		out.EbpfXDPEdge = false
	}
	return out
}

// OpenRTBAllowed gates /openrtb/bid and RTB auction on /track.
func OpenRTBAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.OpenRTBEnabled()
}

// EbpfEdgeAllowed gates edge XDP / edge-bpf-sync.
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
