package campaign

import (
	"strings"
)

const campaignListWarningsBudgetUsedPctMin = 90.0

// CampaignDTOHasListWarning matches campaigns directory row alert rules (budget pressure, margin breach).
func CampaignDTOHasListWarning(dto CampaignDTO) bool {
	if strings.EqualFold(strings.TrimSpace(dto.Status), "PAUSED") ||
		strings.EqualFold(strings.TrimSpace(dto.Status), "ARCHIVED") {
		return false
	}
	if dto.MarginBreach {
		return true
	}
	normalized := strings.TrimSpace(strings.ToUpper(dto.Status))
	if normalized == "EXHAUSTED" || normalized == "ERROR" || normalized == "FAILED" {
		return true
	}
	if dto.BudgetUsedPct != nil && *dto.BudgetUsedPct >= campaignListWarningsBudgetUsedPctMin {
		return true
	}
	return false
}

func normalizeCampaignListStatusFilter(status string) (statusFilter string, warningsOnly bool) {
	if strings.EqualFold(strings.TrimSpace(status), "WARNINGS") {
		return "", true
	}
	return strings.TrimSpace(status), false
}
