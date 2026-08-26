package licensing

type Limits struct {
	MaxRPS              uint64 `json:"max_rps" yaml:"max_rps"`
	MaxRequestsPerDay   uint64 `json:"max_requests_per_day" yaml:"max_requests_per_day"`
	MaxActiveCampaigns  uint64 `json:"max_active_campaigns" yaml:"max_active_campaigns"`
	MaxRegions          uint64 `json:"max_regions" yaml:"max_regions"`
	MaxTenants          uint64 `json:"max_tenants" yaml:"max_tenants"`
	MaxEventsPerMonth   uint64 `json:"max_events_per_month" yaml:"max_events_per_month"`
	MaxAPIKeys          uint64 `json:"max_api_keys" yaml:"max_api_keys"`
	MaxExportChunkBytes uint64 `json:"max_export_chunk_bytes" yaml:"max_export_chunk_bytes"`
	MaxActivations      uint64 `json:"max_activations" yaml:"max_activations"`
	QuotaResetTimezone  string `json:"quota_reset_timezone" yaml:"quota_reset_timezone"`
}

type FeatureSet struct {
	RtbLive                  bool `json:"rtb_live" yaml:"rtb_live"`
	OpenRTBEngine            bool `json:"openrtb_engine" yaml:"openrtb_engine"`
	IvtMLDetector            bool `json:"ivt_ml_detector" yaml:"ivt_ml_detector"`
	EbpfXDPEdge              bool `json:"ebpf_xdp_edge" yaml:"ebpf_xdp_edge"`
	MlFraudBoost             bool `json:"ml_fraud_boost" yaml:"ml_fraud_boost"`
	MultiRegion              bool `json:"multi_region" yaml:"multi_region"`
	SlotMigration            bool `json:"slot_migration" yaml:"slot_migration"`
	MarginGuard              bool `json:"margin_guard" yaml:"margin_guard"`
	ExternalResidentialIntel bool `json:"external_residential_intel" yaml:"external_residential_intel"`
	ModeratorIntelFeed       bool `json:"moderator_intel_feed" yaml:"moderator_intel_feed"`
	AdPlatformCampaignAPI    bool `json:"ad_platform_campaign_api" yaml:"ad_platform_campaign_api"`
}

type Entitlements struct {
	VolumeBand VolumeBand `json:"volume_band,omitempty"`
	Limits     Limits     `json:"limits"`
	Features   FeatureSet `json:"features"`
}

type LimitsDTO struct {
	MaxRPS              uint64 `json:"max_rps"`
	MaxRequestsPerDay   uint64 `json:"max_requests_per_day"`
	MaxActiveCampaigns  uint64 `json:"max_active_campaigns"`
	MaxRegions          uint64 `json:"max_regions"`
	MaxTenants          uint64 `json:"max_tenants"`
	MaxEventsPerMonth   uint64 `json:"max_events_per_month"`
	MaxAPIKeys          uint64 `json:"max_api_keys"`
	MaxExportChunkBytes uint64 `json:"max_export_chunk_bytes"`
	MaxActivations      uint64 `json:"max_activations"`
	QuotaResetTimezone  string `json:"quota_reset_timezone"`
}

type FeatureSetDTO struct {
	RtbLive                  bool `json:"rtb_live"`
	OpenRTBEngine            bool `json:"openrtb_engine"`
	IvtMLDetector            bool `json:"ivt_ml_detector"`
	EbpfXDPEdge              bool `json:"ebpf_xdp_edge"`
	MlFraudBoost             bool `json:"ml_fraud_boost"`
	MultiRegion              bool `json:"multi_region"`
	SlotMigration            bool `json:"slot_migration"`
	MarginGuard              bool `json:"margin_guard"`
	ExternalResidentialIntel bool `json:"external_residential_intel"`
	ModeratorIntelFeed       bool `json:"moderator_intel_feed"`
	AdPlatformCampaignAPI    bool `json:"ad_platform_campaign_api"`
}

type LicenseStatusDTO struct {
	DeploymentID   string        `json:"deployment_id"`
	LicenseID      string        `json:"license_id"`
	Plan           string        `json:"plan"`
	VolumeBand     string        `json:"volume_band"`
	State          string        `json:"state"`
	ValidUntil     string        `json:"valid_until"`
	GraceEndsAt    string        `json:"grace_ends_at,omitempty"`
	Limits         LimitsDTO     `json:"limits"`
	Features       FeatureSetDTO `json:"features"`
	LastVerifiedAt string        `json:"last_verified_at"`
	RefreshMode    string        `json:"refresh_mode"`
	LastRefreshErr string        `json:"last_refresh_error,omitempty"`
	OfflineDays    int           `json:"offline_days,omitempty"`
	BannerSeverity string        `json:"banner_severity,omitempty"`
}
