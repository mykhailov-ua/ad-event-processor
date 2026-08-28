package reports

type auditCampaignFraudChange struct {
	FraudThresholdPass       uint8 `json:"fraud_threshold_pass"`
	FraudThresholdSuspect    uint8 `json:"fraud_threshold_suspect"`
	FraudThresholdIVT        uint8 `json:"fraud_threshold_ivt"`
	FraudThresholdBlock      uint8 `json:"fraud_threshold_block"`
	SilentRejectEnabled      bool  `json:"silent_reject_enabled"`
	BehaviorFlags            int32 `json:"behavior_flags"`
	CanvasRetestEnabled      bool  `json:"canvas_retest_enabled"`
	CgnatIPPolicyEnabled     bool  `json:"cgnat_ip_policy_enabled"`
	AcceptLangGeoEnabled     bool  `json:"accept_lang_geo_enabled"`
	JSONSerializationEnabled bool  `json:"json_serialization_enabled"`
}

type SourceRowDTO struct {
	CampaignID   string  `json:"campaign_id"`
	Sub1         string  `json:"sub1,omitempty"`
	Sub2         string  `json:"sub2,omitempty"`
	Country      string  `json:"country,omitempty"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	CPAMicro     int64   `json:"cpa_micro"`
	ROIPct       float64 `json:"roi_pct"`
	CTR          float64 `json:"ctr"`
	IVTRate      float64 `json:"ivt_rate"`
	QualityScore float64 `json:"quality_score"`
}

type FraudGeoHintDTO struct {
	Country    string  `json:"country"`
	IVTRate    float64 `json:"ivt_rate"`
	IVTEvents  int64   `json:"ivt_events"`
	Clicks     int64   `json:"clicks"`
	CampaignID string  `json:"campaign_id,omitempty"`
}
