package dashboardadmin

import "strings"

func campaignUtilizationPct(spendMicro, budgetMicro int64) float64 {
	if budgetMicro <= 0 {
		return 0
	}
	return float64(spendMicro) / float64(budgetMicro) * 100
}

func estimateDeliveryPct(impressions7d int64, status string) float64 {
	if strings.EqualFold(status, "PAUSED") {
		return 0
	}
	if impressions7d <= 0 {
		return 0
	}
	if impressions7d < 1000 {
		return 35
	}
	if impressions7d < 10000 {
		return 68
	}
	return 92
}

func pacingDriftPct(impressions7d int64, status string) float64 {
	if !strings.EqualFold(status, "ACTIVE") {
		return 0
	}
	return 100 - estimateDeliveryPct(impressions7d, status)
}

func pausedUnderDelivery(impressions7d, budgetMicro int64) bool {
	if budgetMicro <= 0 {
		return false
	}
	return impressions7d > 0 && impressions7d < 1000
}

func overspendRisk(utilizationPct float64, pacingMode, status string) bool {
	if !strings.EqualFold(status, "ACTIVE") {
		return false
	}
	if utilizationPct >= 90 {
		return true
	}
	mode := strings.ToLower(strings.TrimSpace(pacingMode))
	if mode == "asap" && utilizationPct >= 75 {
		return true
	}
	return false
}
