package adminapi

type DashboardServiceCard struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type DashboardSummaryDTO struct {
	GeneratedAt      string                 `json:"generated_at"`
	Services         []DashboardServiceCard `json:"services"`
	DriftMicroMax    float64                `json:"drift_micro_max"`
	DriftAlert       bool                   `json:"drift_alert"`
	RPSEstimate      float64                `json:"rps_estimate"`
	OutboxPending    int64                  `json:"outbox_pending"`
	EmergencyBreaker string                 `json:"emergency_breaker"`
}

type DashboardMetricPoint struct {
	Name       string  `json:"name"`
	LabelsHash string  `json:"labels_hash,omitempty"`
	Timestamp  string  `json:"ts"`
	Value      float64 `json:"value"`
}

type DashboardMetricsDTO struct {
	Range       string                 `json:"range"`
	BucketSec   int                    `json:"bucket_sec"`
	Points      []DashboardMetricPoint `json:"points"`
	GeneratedAt string                 `json:"generated_at"`
}
