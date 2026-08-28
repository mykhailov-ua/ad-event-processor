package licensing

func (f FeatureSet) Normalized() FeatureSet {
	out := f
	if out.OpenRTBEngine || out.RtbLive {
		out.OpenRTBEngine = true
		out.RtbLive = true
	}
	if out.MlFraudBoost {
		out.MlFraudBoost = true
	}
	return out
}

func (f FeatureSet) OpenRTBEnabled() bool {
	n := f.Normalized()
	return n.OpenRTBEngine || n.RtbLive
}

func (f FeatureSet) MlFraudBoostEnabled() bool {
	return f.Normalized().MlFraudBoost
}

func (f FeatureSet) IvtMLEnabled() bool {
	return f.Normalized().IvtMLDetector
}

func (f FeatureSet) EbpfEdgeEnabled() bool {
	return f.Normalized().EbpfXDPEdge
}

func (f FeatureSet) MultiRegionEnabled() bool {
	return f.Normalized().MultiRegion
}

func (f FeatureSet) ExternalResidentialIntelEnabled() bool {
	return f.Normalized().ExternalResidentialIntel
}

func (f FeatureSet) ModeratorIntelFeedEnabled() bool {
	return f.Normalized().ModeratorIntelFeed
}

func (f FeatureSet) AdPlatformCampaignAPIEnabled() bool {
	return f.Normalized().AdPlatformCampaignAPI
}

func (f FeatureSet) FraudDisputeEvidenceEnabled() bool {
	return f.Normalized().FraudDisputeEvidence
}
