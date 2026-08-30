package commandpalette

import "github.com/google/uuid"

type ItemDTO struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	StatusLabel string `json:"status_label,omitempty"`
	StatusTone  string `json:"status_tone,omitempty"`
	Href        string `json:"href"`
	Meta        string `json:"meta,omitempty"`
	Group       string `json:"group,omitempty"`
}

type SearchResponse struct {
	Items          []ItemDTO `json:"items"`
	Total          int64     `json:"total"`
	Limit          int       `json:"limit"`
	Degraded       bool      `json:"degraded"`
	FreshnessLabel string    `json:"freshness_label,omitempty"`
}

type RoutesResponse struct {
	Items []ItemDTO `json:"items"`
	Total int       `json:"total"`
}

type RecentsResponse struct {
	Items []ItemDTO `json:"items"`
	Total int       `json:"total"`
}

type RecordRecentRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Item       ItemDTO   `json:"item"`
}

type navEntry struct {
	ID           string
	Kind         string
	Label        string
	Href         string
	Meta         string
	Group        string
	Permissions  []string
	LicenseGated bool
	FeatureKey   string
	ReportKey    string
}
