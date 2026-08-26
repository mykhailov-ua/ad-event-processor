package automation

import (
	"fmt"
	"strings"
)

var validMetrics = map[string]struct{}{
	"clicks":             {},
	"spend_micro":        {},
	"roi_pct":            {},
	"cr":                 {},
	"fraud_reject_count": {},
	"fraud_reject_rate":  {},
	"ivt_rate":           {},
	"silent_reject_rate": {},
}

var validOperators = map[string]struct{}{
	"gt":  {},
	"lt":  {},
	"gte": {},
	"lte": {},
}

func canonicalAutomationMetric(metric string) string {
	switch metric {
	case "ivt_rate":
		return "fraud_reject_rate"
	default:
		return metric
	}
}

func NormalizeMetric(metric string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(metric))
	m = canonicalAutomationMetric(m)
	if _, ok := validMetrics[m]; !ok {
		return "", fmt.Errorf("invalid metric %q", metric)
	}
	return m, nil
}

func NormalizeOperator(op string) (string, error) {
	o := strings.ToLower(strings.TrimSpace(op))
	if _, ok := validOperators[o]; !ok {
		return "", fmt.Errorf("invalid operator %q", op)
	}
	return o, nil
}

func NormalizeGroupBy(groupBy string) (string, error) {
	g := strings.ToLower(strings.TrimSpace(groupBy))
	if g == "" {
		return GroupByPlacement, nil
	}
	switch g {
	case GroupByPlacement, GroupByCampaign:
		return g, nil
	default:
		return "", fmt.Errorf("invalid group_by %q", groupBy)
	}
}

func ThresholdBreached(operator string, observed, threshold float64) bool {
	switch operator {
	case "gt":
		return observed > threshold
	case "gte":
		return observed >= threshold
	case "lt":
		return observed < threshold
	case "lte":
		return observed <= threshold
	default:
		return false
	}
}

func CalcROIPct(profitMicro, spendMicro int64) float64 {
	if spendMicro <= 0 {
		return 0
	}
	return float64(profitMicro) / float64(spendMicro) * 100
}

func CalcCRPct(clicks, conversions uint64) float64 {
	if clicks == 0 {
		return 0
	}
	return float64(conversions) / float64(clicks) * 100
}

func CalcFraudRejectRatePct(clicks, fraudRejectCount uint64) float64 {
	if clicks == 0 {
		return 0
	}
	return float64(fraudRejectCount) / float64(clicks) * 100
}

// CalcIVTRatePct is a legacy name for fraud stream hard-reject rate (not fraud tier IVT).
func CalcIVTRatePct(clicks, fraudRejectCount uint64) float64 {
	return CalcFraudRejectRatePct(clicks, fraudRejectCount)
}

func CalcSilentRejectRatePct(clicks, silentRejectCount uint64) float64 {
	if clicks == 0 {
		return 0
	}
	return float64(silentRejectCount) / float64(clicks) * 100
}

func ClampWindowMinutes(window int) int {
	if window < 15 {
		return 15
	}
	if window > 1440 {
		return 1440
	}
	return window
}

func ClampCooldownMinutes(cooldown int) int {
	if cooldown < 15 {
		return 15
	}
	if cooldown > 10080 {
		return 10080
	}
	return cooldown
}
