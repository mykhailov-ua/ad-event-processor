package controlplane

import (
	"fmt"
	"time"

	"espx/internal/controlplane/adminapi"
)

func enrichBuyerPortfolioCommercial(resp *BuyerPortfolioDTO) {
	if resp == nil {
		return
	}
	now := time.Now().UTC()
	resp.Recommendations = make([]adminapi.RecommendationCardDTO, 0, 4)
	resp.Alerts = make([]adminapi.AlertCardDTO, 0, 4)

	for _, c := range resp.Campaigns {
		if c.OverspendRisk {
			resp.Recommendations = append(resp.Recommendations, adminapi.RecommendationCardDTO{
				ID:          fmt.Sprintf("pause-%s", c.ID),
				Type:        "pause_overspend",
				CampaignID:  c.ID,
				Title:       "Pause overspending campaign",
				Detail:      fmt.Sprintf("%s is at %.0f%% budget utilization", c.Name, c.UtilizationPct),
				Confidence:  0.85,
				ImpactMicro: c.SpendMicro / 10,
				CreatedAt:   now.Format(time.RFC3339),
				Actions: []adminapi.ActionDTO{
					{ID: "pause", Label: "Pause campaign", RequiresConfirm: true},
				},
			})
			resp.Alerts = append(resp.Alerts, adminapi.AlertCardDTO{
				ID:     fmt.Sprintf("overspend-%s", c.ID),
				Level:  "warning",
				Title:  "Overspend risk",
				Detail: c.Name,
				Route:  "/campaigns/" + c.ID,
			})
		}
		if c.Status == "PAUSED" && pausedUnderDelivery(c.Impressions7d, c.BudgetMicro) {
			resp.Recommendations = append(resp.Recommendations, adminapi.RecommendationCardDTO{
				ID:         fmt.Sprintf("resume-%s", c.ID),
				Type:       "resume_under_delivery",
				CampaignID: c.ID,
				Title:      "Resume under-delivering campaign",
				Detail:     fmt.Sprintf("%s had low delivery before pause (%d impr/7d)", c.Name, c.Impressions7d),
				Confidence: 0.7,
				CreatedAt:  now.Format(time.RFC3339),
				Actions: []adminapi.ActionDTO{
					{ID: "resume", Label: "Resume campaign", RequiresConfirm: true},
				},
			})
		}
		if c.Status == "ACTIVE" && c.UtilizationPct < 40 && c.BudgetMicro > 0 {
			resp.Recommendations = append(resp.Recommendations, adminapi.RecommendationCardDTO{
				ID:          fmt.Sprintf("budget-%s", c.ID),
				Type:        "budget_bump",
				CampaignID:  c.ID,
				Title:       "Consider budget increase",
				Detail:      fmt.Sprintf("%s has headroom at %.0f%% utilization", c.Name, c.UtilizationPct),
				Confidence:  0.55,
				ImpactMicro: c.BudgetMicro / 5,
				CreatedAt:   now.Format(time.RFC3339),
				Actions: []adminapi.ActionDTO{
					{ID: "edit_budget", Label: "Open campaign", RequiresConfirm: false},
				},
			})
		}
	}
	if resp.OverspendCount > 0 {
		resp.Alerts = append(resp.Alerts, adminapi.AlertCardDTO{
			ID:     "portfolio-overspend",
			Level:  "critical",
			Title:  "Portfolio overspend",
			Detail: fmt.Sprintf("%d campaigns need attention", resp.OverspendCount),
			Route:  "/campaigns/portfolio",
		})
	}
	if len(resp.Recommendations) > 6 {
		resp.Recommendations = resp.Recommendations[:6]
	}
	if len(resp.Alerts) > 6 {
		resp.Alerts = resp.Alerts[:6]
	}
}
