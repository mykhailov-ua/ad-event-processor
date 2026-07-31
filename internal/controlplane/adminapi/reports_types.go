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
