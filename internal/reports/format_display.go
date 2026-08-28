package reports

import (
	"fmt"
	"math"

	"ad-event-processor/pkg/money"
)

func formatMicro(m int64) string {
	return money.FormatFixed2(m)
}

func formatRateDisplay(rate float64) string {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", rate*100)
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
