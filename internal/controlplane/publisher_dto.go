package controlplane

import "github.com/google/uuid"

type PublisherBind struct {
	SellerID           string
	PublisherAccountID string
	CustomerID         uuid.UUID
}

type PublisherKPIsDTO struct {
	Impressions int64   `json:"impressions"`
	FillRate    float64 `json:"fill_rate"`
	EcpmMicro   int64   `json:"ecpm_micro"`
	IVTRate     float64 `json:"ivt_rate"`
}

type PublisherPlacementDTO struct {
	PlacementID  string  `json:"placement_id"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	FillRate     float64 `json:"fill_rate"`
	RevenueMicro int64   `json:"revenue_micro"`
	EcpmMicro    int64   `json:"ecpm_micro"`
}

type PublisherDashboardDTO struct {
	SellerID           string                  `json:"seller_id"`
	PublisherAccountID string                  `json:"publisher_account_id,omitempty"`
	From               string                  `json:"from"`
	To                 string                  `json:"to"`
	KPIs               PublisherKPIsDTO        `json:"kpis"`
	Placements         []PublisherPlacementDTO `json:"placements"`
}

type PublisherStatementDTO struct {
	ID              int64  `json:"id"`
	AmountMicro     int64  `json:"amount_micro"`
	CreatedAt       string `json:"created_at"`
	CampaignID      string `json:"campaign_id,omitempty"`
	IdempotencyHash string `json:"idempotency_hash,omitempty"`
}

type PublisherStatementListResponse struct {
	Items []PublisherStatementDTO `json:"items"`
	Total int64                   `json:"total"`
}
