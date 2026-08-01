package adminapi

type TableDTO struct {
	Columns    []ColumnDTO      `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	Totals     map[string]any   `json:"totals,omitempty"`
	Freshness  DataFreshnessDTO `json:"freshness"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type PlacementReportRowDTO struct {
	PlacementID  string  `json:"placement_id"`
	CampaignID   string  `json:"campaign_id"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	ROIPct       float64 `json:"roi_pct"`
	CPAMicro     int64   `json:"cpa_micro"`
}

type PlacementReportResponse struct {
	Rows       []PlacementReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO        `json:"freshness"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type KeywordReportRowDTO struct {
	Keyword      string  `json:"keyword"`
	CampaignID   string  `json:"campaign_id"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	ROIPct       float64 `json:"roi_pct"`
}

type KeywordReportResponse struct {
	Rows       []KeywordReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type ColumnDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"`
}

type UnitEconomicsRowDTO struct {
	CampaignID   string  `json:"campaign_id"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	Conversions  int64   `json:"conversions"`
	CPAMicro     int64   `json:"cpa_micro"`
	CPCMicro     int64   `json:"cpc_micro"`
	CPMMicro     int64   `json:"cpm_micro"`
	ROIPct       float64 `json:"roi_pct"`
	EPCMicro     int64   `json:"epc_micro"`
}

type PivotTableDTO struct {
	RowDim string    `json:"row_dim"`
	ColDim string    `json:"col_dim"`
	Rows   []string  `json:"rows"`
	Cols   []string  `json:"cols"`
	Cells  [][][]any `json:"cells"`
}

type PostbackReconRowDTO struct {
	ClickID             string `json:"click_id"`
	CampaignID          string `json:"campaign_id"`
	ExpectedPayoutMicro int64  `json:"expected_payout_micro"`
	RecordedPayoutMicro int64  `json:"recorded_payout_micro"`
	DeltaPayoutMicro    int64  `json:"delta_payout_micro"`
	AttributionLagSec   int64  `json:"attribution_lag_sec"`
	Status              string `json:"status"`
}

type ReportJobSpec struct {
	ReportKey  string         `json:"report_key"`
	CustomerID string         `json:"customer_id"`
	Period     PeriodDTO      `json:"period"`
	Compare    *PeriodDTO     `json:"compare,omitempty"`
	GroupBy    []string       `json:"group_by,omitempty"`
	Filters    map[string]any `json:"filters,omitempty"`
	Format     string         `json:"format"`
}

type CampaignMetricsDTO struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	Conversions int64 `json:"conversions"`
}

type CampaignHourlyBucketDTO struct {
	Hour        string `json:"hour"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignStatsDTO struct {
	CampaignID   string                    `json:"campaign_id"`
	CurrentSpend string                    `json:"current_spend"`
	Metrics      CampaignMetricsDTO        `json:"metrics"`
	Hourly       []CampaignHourlyBucketDTO `json:"hourly"`
	Granularity  string                    `json:"granularity"`
	From         string                    `json:"from"`
	To           string                    `json:"to"`
	Stale        bool                      `json:"stale"`
	Source       string                    `json:"source"`
	Consistency  string                    `json:"consistency"`
}

const (
	MetricSpendMicro     = "spend_micro"
	MetricRevenueMicro   = "revenue_micro"
	MetricProfitMicro    = "profit_micro"
	MetricROIPct         = "roi_pct"
	MetricCPAMicro       = "cpa_micro"
	MetricCPCMicro       = "cpc_micro"
	MetricCPMMicro       = "cpm_micro"
	MetricCTR            = "ctr"
	MetricEPCMicro       = "epc_micro"
	MetricIVTRate        = "ivt_rate"
	MetricUtilizationPct = "utilization_pct"
	MetricAvailableMicro = "available_micro"
	MetricPacingDriftPct = "pacing_drift_pct"
)

var MetricFormulas = map[string]string{
	MetricSpendMicro:     "SUM(ledger debits) or CH cost",
	MetricRevenueMicro:   "SUM(postback payout)",
	MetricProfitMicro:    "revenue_micro - spend_micro",
	MetricROIPct:         "profit_micro / spend_micro * 100",
	MetricCPAMicro:       "spend_micro / conversions",
	MetricIVTRate:        "ivt / clicks",
	MetricUtilizationPct: "current_spend / budget_limit",
	MetricAvailableMicro: "balance + overdraft - reserved",
	MetricPacingDriftPct: "(actual - planned) / planned",
}
