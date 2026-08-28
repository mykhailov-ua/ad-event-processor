package reports

type DataFreshnessDTO struct {
	AsOf         string                   `json:"as_of"`
	Consistency  string                   `json:"consistency"`
	Stale        bool                     `json:"stale"`
	CHLagSeconds int                      `json:"ch_lag_seconds,omitempty"`
	Sources      []DataSourceFreshnessDTO `json:"sources,omitempty"`
}
