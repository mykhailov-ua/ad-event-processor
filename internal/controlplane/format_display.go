package controlplane

import (
	"fmt"
	"math"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/pkg/money"
)

func redactedMoneyDisplay() string {
	return "—"
}

func moneyDisplayFromDecimal(amount string) string {
	if amount == "" {
		return ""
	}
	return "USD " + amount
}

func attachCampaignMoneyDisplay(dto *CampaignDTO) {
	campaign.AttachCampaignMoneyDisplay(dto)
}

func formatMicrosDisplay(micros int64, currency string) string {
	if currency == "" {
		currency = "USD"
	}
	return fmt.Sprintf("%s %s", currency, money.FormatFixed2(micros))
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

func formatShareLabel(sharePct float64) string {
	if math.IsNaN(sharePct) || math.IsInf(sharePct, 0) {
		return "0.0% of filtered events"
	}
	return fmt.Sprintf("%.1f%% of filtered events", sharePct)
}

func formatDeltaLabel(deltaPct float64) string {
	if math.IsNaN(deltaPct) || math.IsInf(deltaPct, 0) || deltaPct == 0 {
		return "0.0% vs prior period"
	}
	sign := "+"
	if deltaPct < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f%% vs prior period", sign, deltaPct)
}

func deltaTone(deltaPct float64) string {
	if deltaPct > 0.0001 {
		return "positive"
	}
	if deltaPct < -0.0001 {
		return "negative"
	}
	return "neutral"
}

func campaignStatusLabel(status string) string {
	return campaign.CampaignStatusLabel(status)
}

func campaignStatusTone(status string) string {
	return campaign.CampaignStatusTone(status)
}
