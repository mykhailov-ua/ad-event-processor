package controlplane

import (
	"context"
	"encoding/csv"
	"fmt"

	"github.com/google/uuid"
)

func writeCampaignOverviewCSV(ctx context.Context, w *csv.Writer, deps ReportExportDeps, customerID string) error {
	if deps.BuyerPortfolio == nil {
		return fmt.Errorf("portfolio reader not configured")
	}
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("invalid customer_id")
	}
	portfolio, err := deps.BuyerPortfolio.GetBuyerPortfolio(ctx, cid)
	if err != nil {
		return err
	}
	if err := w.Write([]string{"campaign_id", "name", "status", "impressions_7d", "clicks_7d", "utilization_pct", "pacing_drift_pct", "overspend_risk"}); err != nil {
		return err
	}
	for _, c := range portfolio.Campaigns {
		if err := w.Write([]string{
			c.ID, c.Name, c.Status,
			fmt.Sprintf("%d", c.Impressions7d), fmt.Sprintf("%d", c.Clicks7d),
			fmt.Sprintf("%.4f", c.UtilizationPct), fmt.Sprintf("%.4f", c.PacingDriftPct),
			fmt.Sprintf("%t", c.OverspendRisk),
		}); err != nil {
			return err
		}
	}
	return nil
}

func writeCustomerPortfolioCSV(ctx context.Context, w *csv.Writer, deps ReportExportDeps, customerID string) error {
	if deps.BuyerPortfolio == nil {
		return fmt.Errorf("portfolio reader not configured")
	}
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("invalid customer_id")
	}
	portfolio, err := deps.BuyerPortfolio.GetBuyerPortfolio(ctx, cid)
	if err != nil {
		return err
	}
	if err := w.Write([]string{
		"row_type", "campaign_id", "name", "status", "active", "paused", "archived",
		"impressions_7d", "clicks_7d", "overspend_count", "attention_count", "campaigns_sample",
		"utilization_pct", "pacing_drift_pct", "overspend_risk",
	}); err != nil {
		return err
	}
	if err := w.Write([]string{
		"summary", "", "", "",
		fmt.Sprintf("%d", portfolio.Active), fmt.Sprintf("%d", portfolio.Paused), fmt.Sprintf("%d", portfolio.Archived),
		fmt.Sprintf("%d", portfolio.Impressions7d), fmt.Sprintf("%d", portfolio.Clicks7d),
		fmt.Sprintf("%d", portfolio.OverspendCount), fmt.Sprintf("%d", len(portfolio.Attention)), fmt.Sprintf("%d", len(portfolio.Campaigns)),
		"", "", "",
	}); err != nil {
		return err
	}
	for _, c := range portfolio.Campaigns {
		if err := w.Write([]string{
			"campaign", c.ID, c.Name, c.Status, "", "", "",
			fmt.Sprintf("%d", c.Impressions7d), fmt.Sprintf("%d", c.Clicks7d),
			"", "", "",
			fmt.Sprintf("%.4f", c.UtilizationPct), fmt.Sprintf("%.4f", c.PacingDriftPct),
			fmt.Sprintf("%t", c.OverspendRisk),
		}); err != nil {
			return err
		}
	}
	return nil
}
