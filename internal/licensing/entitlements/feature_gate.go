package entitlements

import "strings"

func FeatureAllowedByKey(state LicenseState, ent Entitlements, featureKey string) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	f := ent.Features.Normalized()
	switch strings.TrimSpace(featureKey) {
	case "openrtb", "rtb_live", "openrtb_engine":
		return f.OpenRTBEnabled()
	case "fraud_dispute_evidence":
		return f.FraudDisputeEvidenceEnabled()
	case "ad_platform_campaign_api":
		return f.AdPlatformCampaignAPIEnabled()
	case "margin_guard":
		return f.MarginGuardEnabled()
	case "slot_migration":
		return f.SlotMigrationEnabled()
	case "ebpf_xdp_edge":
		return f.EbpfEdgeEnabled()
	case "ml_fraud_boost":
		return f.MlFraudBoostEnabled()
	case "ivt_ml_detector":
		return f.IvtMLEnabled()
	case "multi_region":
		return f.MultiRegionEnabled()
	case "external_residential_intel":
		return f.ExternalResidentialIntelEnabled()
	case "moderator_intel_feed":
		return f.ModeratorIntelFeedEnabled()
	case "broker_wal":
		return f.BrokerWalEnabled()
	default:
		return false
	}
}

func MarginGuardAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.MarginGuardEnabled()
}

func SlotMigrationAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.SlotMigrationEnabled()
}

func AdPlatformCampaignAPIAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.AdPlatformCampaignAPIEnabled()
}

func BrokerWalAllowed(state LicenseState, ent Entitlements) bool {
	if state == StateExpired || state == StateRevoked {
		return false
	}
	return ent.Features.BrokerWalEnabled()
}
