package licensing

func MergeLimits(dst *Limits, src Limits) {
	if src.MaxRPS != 0 {
		dst.MaxRPS = src.MaxRPS
	}
	if src.MaxRequestsPerDay != 0 {
		dst.MaxRequestsPerDay = src.MaxRequestsPerDay
	}
	if src.MaxActiveCampaigns != 0 {
		dst.MaxActiveCampaigns = src.MaxActiveCampaigns
	}
	if src.MaxRegions != 0 {
		dst.MaxRegions = src.MaxRegions
	}
	if src.MaxTenants != 0 {
		dst.MaxTenants = src.MaxTenants
	}
	if src.MaxEventsPerMonth != 0 {
		dst.MaxEventsPerMonth = src.MaxEventsPerMonth
	}
	if src.MaxAPIKeys != 0 {
		dst.MaxAPIKeys = src.MaxAPIKeys
	}
	if src.MaxExportChunkBytes != 0 {
		dst.MaxExportChunkBytes = src.MaxExportChunkBytes
	}
	if src.MaxActivations != 0 {
		dst.MaxActivations = src.MaxActivations
	}
	if src.QuotaResetTimezone != "" {
		dst.QuotaResetTimezone = src.QuotaResetTimezone
	}
}

func MergeFeatures(dst *FeatureSet, src FeatureSet) {
	src = src.Normalized()
	dst.RtbLive = src.RtbLive
	dst.OpenRTBEngine = src.OpenRTBEngine
	dst.IvtMLDetector = src.IvtMLDetector
	dst.EbpfXDPEdge = src.EbpfXDPEdge
	dst.MlFraudBoost = src.MlFraudBoost
	dst.MultiRegion = src.MultiRegion
	dst.SlotMigration = src.SlotMigration
	dst.MarginGuard = src.MarginGuard
	dst.ExternalResidentialIntel = src.ExternalResidentialIntel
}
