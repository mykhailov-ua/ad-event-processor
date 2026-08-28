package campaign

import (
	"fmt"
	"math"
)

func attachCampaignMoneyDisplay(dto *CampaignDTO) {
	AttachCampaignMoneyDisplay(dto)
}

func AttachCampaignMoneyDisplay(dto *CampaignDTO) {
	if dto == nil {
		return
	}
	if dto.BudgetLimitDisplay == "" && dto.BudgetLimit != "" {
		dto.BudgetLimitDisplay = moneyDisplayFromDecimal(dto.BudgetLimit)
	}
	if dto.DailyBudgetDisplay == "" && dto.DailyBudget != "" {
		dto.DailyBudgetDisplay = moneyDisplayFromDecimal(dto.DailyBudget)
	}
	if dto.CurrentSpendDisplay == "" && dto.CurrentSpend != "" {
		dto.CurrentSpendDisplay = moneyDisplayFromDecimal(dto.CurrentSpend)
	}
}

func moneyDisplayFromDecimal(amount string) string {
	if amount == "" {
		return ""
	}
	return "USD " + amount
}

func formatRateDisplay(rate float64) string {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", rate*100)
}

func formatBasisPointsDisplay(bps int) string {
	return formatRateDisplay(float64(bps) / 10000)
}

func campaignStatusLabel(status string) string {
	return CampaignStatusLabel(status)
}

func CampaignStatusLabel(status string) string {
	switch status {
	case "ACTIVE":
		return "Active"
	case "PAUSED":
		return "Paused"
	case "ARCHIVED":
		return "Archived"
	default:
		if status == "" {
			return "Unknown"
		}
		return status
	}
}

func campaignStatusTone(status string) string {
	return CampaignStatusTone(status)
}

func CampaignStatusTone(status string) string {
	switch status {
	case "ACTIVE":
		return "success"
	case "PAUSED":
		return "warning"
	case "ARCHIVED":
		return "muted"
	default:
		return "muted"
	}
}
