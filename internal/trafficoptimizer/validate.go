package trafficoptimizer

import (
	"fmt"
	"strings"
	"time"
)

var allowedEvalIntervalMinutes = []int{5, 10, 15, 30, 60}

func NormalizeScope(scope string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(scope))
	switch s {
	case ScopeLander, ScopeOffer, ScopeCreative:
		return s, nil
	case "":
		return "", fmt.Errorf("scope is required")
	default:
		return "", fmt.Errorf("invalid scope %q", scope)
	}
}

func NormalizeObjective(objective string) (string, error) {
	o := strings.ToLower(strings.TrimSpace(objective))
	switch o {
	case ObjectiveCR, ObjectiveEPC, ObjectiveRevenue, ObjectiveROI:
		return o, nil
	case "":
		return "", fmt.Errorf("objective is required")
	default:
		return "", fmt.Errorf("invalid objective %q", objective)
	}
}

func NormalizeAlgorithm(algorithm string) (string, error) {
	a := strings.ToLower(strings.TrimSpace(algorithm))
	switch a {
	case AlgorithmThompson, AlgorithmProportional:
		return a, nil
	case "":
		return AlgorithmThompson, nil
	default:
		return "", fmt.Errorf("invalid algorithm %q", algorithm)
	}
}

func ValidateObjectiveAlgorithmPair(objective, algorithm string) error {
	switch objective {
	case ObjectiveCR:
		if algorithm != AlgorithmThompson {
			return fmt.Errorf("objective %q requires algorithm %q", objective, AlgorithmThompson)
		}
	case ObjectiveEPC, ObjectiveRevenue, ObjectiveROI:
		if algorithm != AlgorithmProportional {
			return fmt.Errorf("objective %q requires algorithm %q", objective, AlgorithmProportional)
		}
	default:
		return fmt.Errorf("objective %q not supported", objective)
	}
	return nil
}

func ValidateRuleTargets(scope, brandID, campaignID, flowID string) error {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == ScopeCreative {
		if strings.TrimSpace(brandID) == "" {
			return fmt.Errorf("brand_id is required for creative scope")
		}
		return nil
	}
	if strings.TrimSpace(brandID) != "" {
		return fmt.Errorf("brand_id is only valid for creative scope")
	}
	_ = campaignID
	_ = flowID
	return nil
}

const (
	minRuleClicks                    = 100
	maxLookbackMinutesWithoutLicense = 10080
)

func ValidateMinClicks(minClicks int) error {
	if minClicks < minRuleClicks {
		return fmt.Errorf("min_clicks must be at least %d", minRuleClicks)
	}
	return nil
}

func ValidateMinSpendMicro(objective string, minSpendMicro int64) error {
	if objective == ObjectiveROI && minSpendMicro <= 0 {
		return fmt.Errorf("min_spend_micro must be positive for roi objective")
	}
	return nil
}

func ValidateLookbackMinutes(lookback int, allowExtended bool) error {
	if lookback > maxLookbackMinutesWithoutLicense && !allowExtended {
		return fmt.Errorf("lookback_minutes above %d requires traffic_optimizer license", maxLookbackMinutesWithoutLicense)
	}
	return nil
}

func NormalizeEvalIntervalMinutes(requested, floor int) (int, error) {
	if requested == 0 {
		requested = 15
	}
	if floor <= 0 {
		floor = 15
	}
	if requested < floor {
		return 0, fmt.Errorf("eval_interval_minutes must be at least %d", floor)
	}
	for _, v := range allowedEvalIntervalMinutes {
		if requested == v {
			return requested, nil
		}
	}
	return 0, fmt.Errorf("eval_interval_minutes must be one of 5, 10, 15, 30, 60")
}

func ClampLookbackMinutes(lookback int) int {
	if lookback < 60 {
		return 60
	}
	if lookback > 10080 {
		return 10080
	}
	return lookback
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

func ClampMaxWeightDeltaPct(pct int) int {
	if pct <= 0 {
		return 50
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func RuleDueForEval(now, lastEvaluatedAt time.Time, hasLastEvaluated bool, evalIntervalMinutes int) bool {
	if evalIntervalMinutes <= 0 {
		evalIntervalMinutes = 15
	}
	if !hasLastEvaluated {
		return true
	}
	return now.Sub(lastEvaluatedAt) >= time.Duration(evalIntervalMinutes)*time.Minute
}
