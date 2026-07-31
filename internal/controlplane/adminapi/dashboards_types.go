package adminapi

type PeriodDTO struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone,omitempty"`
}

type DataFreshnessDTO struct {
	AsOf         string `json:"as_of"`
	Consistency  string `json:"consistency"`
	Stale        bool   `json:"stale"`
	CHLagSeconds int    `json:"ch_lag_seconds,omitempty"`
}

type MetricsBlockDTO struct {
	SpendMicro   int64            `json:"spend_micro"`
	RevenueMicro int64            `json:"revenue_micro"`
	ProfitMicro  int64            `json:"profit_micro"`
	Conversions  int64            `json:"conversions"`
	CPAMicro     int64            `json:"cpa_micro"`
	ROIPct       float64          `json:"roi_pct"`
	Freshness    DataFreshnessDTO `json:"freshness"`
}

type ActionDTO struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	RequiresConfirm bool   `json:"requires_confirm"`
	ImpactMicro     int64  `json:"impact_micro,omitempty"`
}

type SourceRowDTO struct {
	CampaignID   string      `json:"campaign_id"`
	Sub1         string      `json:"sub1,omitempty"`
	Sub2         string      `json:"sub2,omitempty"`
	Country      string      `json:"country,omitempty"`
	Impressions  int64       `json:"impressions"`
	Clicks       int64       `json:"clicks"`
	Conversions  int64       `json:"conversions"`
	SpendMicro   int64       `json:"spend_micro"`
	RevenueMicro int64       `json:"revenue_micro"`
	ProfitMicro  int64       `json:"profit_micro"`
	CPAMicro     int64       `json:"cpa_micro"`
	ROIPct       float64     `json:"roi_pct"`
	CTR          float64     `json:"ctr"`
	IVTRate      float64     `json:"ivt_rate"`
	QualityScore float64     `json:"quality_score"`
	Actions      []ActionDTO `json:"actions,omitempty"`
}

type BuyerCampaignRowDTO struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Status         string      `json:"status"`
	SpendMicro     int64       `json:"spend_micro"`
	BudgetMicro    int64       `json:"budget_micro"`
	UtilizationPct float64     `json:"utilization_pct"`
	ROIPct         float64     `json:"roi_pct"`
	PacingDriftPct float64     `json:"pacing_drift_pct"`
	OverspendRisk  bool        `json:"overspend_risk"`
	Actions        []ActionDTO `json:"actions,omitempty"`
}

type RecommendationCardDTO struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	CampaignID  string      `json:"campaign_id,omitempty"`
	Title       string      `json:"title"`
	Detail      string      `json:"detail"`
	Confidence  float64     `json:"confidence"`
	ImpactMicro int64       `json:"impact_micro"`
	CreatedAt   string      `json:"created_at"`
	ExpiresAt   string      `json:"expires_at"`
	Actions     []ActionDTO `json:"actions,omitempty"`
}

type AlertCardDTO struct {
	ID     string `json:"id"`
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Route  string `json:"route,omitempty"`
}

type BuyerDashboardDTO struct {
	CustomerID      string                  `json:"customer_id"`
	Period          PeriodDTO               `json:"period"`
	KPIs            MetricsBlockDTO         `json:"kpis"`
	Campaigns       []BuyerCampaignRowDTO   `json:"campaigns"`
	TopSources      []SourceRowDTO          `json:"top_sources"`
	WorstSources    []SourceRowDTO          `json:"worst_sources"`
	Alerts          []AlertCardDTO          `json:"alerts"`
	Recommendations []RecommendationCardDTO `json:"recommendations"`
}

type AccountantCloseDTO struct {
	CustomerID            string `json:"customer_id"`
	BillingMonth          string `json:"billing_month"`
	InvariantOK           bool   `json:"invariant_ok"`
	InvariantDeltaMicro   int64  `json:"invariant_delta_micro"`
	UnreconciledPostbacks int    `json:"unreconciled_postbacks"`
}

type CFOSummaryDTO struct {
	CustomerID string          `json:"customer_id"`
	Period     PeriodDTO       `json:"period"`
	KPIs       MetricsBlockDTO `json:"unit_economics"`
}

type AdOpsHealthDTO struct {
	CustomerID string    `json:"customer_id"`
	Period     PeriodDTO `json:"period"`
}

type FraudOverviewDTO struct {
	CustomerID string    `json:"customer_id"`
	Period     PeriodDTO `json:"period"`
}

type OperatorDashboardDTO struct {
	Period PeriodDTO   `json:"period"`
	XDP    XDPPanelDTO `json:"xdp"`
}

type XDPPanelDTO struct {
	UpdatedAt     string            `json:"updated_at,omitempty"`
	Pass          uint64            `json:"pass"`
	PassAllowlist uint64            `json:"pass_allowlist"`
	Fingerprints  uint64            `json:"fingerprints"`
	Drops         map[string]uint64 `json:"drops"`
}

type CampaignDashboardDTO struct {
	CampaignID string           `json:"campaign_id"`
	KPIs       MetricsBlockDTO  `json:"kpis"`
	Freshness  DataFreshnessDTO `json:"freshness"`
}
