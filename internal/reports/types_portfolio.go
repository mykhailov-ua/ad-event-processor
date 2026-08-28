package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BuyerAttentionDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type BuyerCampaignPortfolioRowDTO struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Status                  string  `json:"status"`
	PacingMode              string  `json:"pacing_mode"`
	Impressions7d           int64   `json:"impressions_7d"`
	Clicks7d                int64   `json:"clicks_7d"`
	SpendMicro              int64   `json:"spend_micro,omitempty"`
	BudgetMicro             int64   `json:"budget_micro,omitempty"`
	UtilizationPct          float64 `json:"utilization_pct,omitempty"`
	PacingDriftPct          float64 `json:"pacing_drift_pct,omitempty"`
	EstimatedPacingDriftPct float64 `json:"estimated_pacing_drift_pct,omitempty"`
	OverspendRisk           bool    `json:"overspend_risk,omitempty"`
	MarginBreach            bool    `json:"margin_breach,omitempty"`
}

type BuyerPortfolioDTO struct {
	CustomerID     string                         `json:"customer_id"`
	Active         int                            `json:"active"`
	Paused         int                            `json:"paused"`
	Archived       int                            `json:"archived"`
	Impressions7d  int64                          `json:"impressions_7d"`
	Clicks7d       int64                          `json:"clicks_7d"`
	OverspendCount int                            `json:"overspend_count,omitempty"`
	Attention      []BuyerAttentionDTO            `json:"attention"`
	Campaigns      []BuyerCampaignPortfolioRowDTO `json:"campaigns"`
	Fraud          *CustomerFraudOverviewDTO      `json:"fraud,omitempty"`
}

type BuyerPortfolioReader interface {
	GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error)
	GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, from, to time.Time) (BuyerPortfolioDTO, error)
}
