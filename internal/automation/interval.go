package automation

import (
	"fmt"
	"time"
)

var AllowedEvalIntervalMinutes = []int{5, 10, 15, 30, 60}

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
	allowed := false
	for _, v := range AllowedEvalIntervalMinutes {
		if requested == v {
			allowed = true
			break
		}
	}
	if !allowed {
		return 0, fmt.Errorf("eval_interval_minutes must be one of 5, 10, 15, 30, 60")
	}
	return requested, nil
}

func RuleDueForEval(now time.Time, lastEvaluatedAt time.Time, hasLastEvaluated bool, evalIntervalMinutes int) bool {
	if evalIntervalMinutes <= 0 {
		evalIntervalMinutes = 15
	}
	if !hasLastEvaluated {
		return true
	}
	return now.Sub(lastEvaluatedAt) >= time.Duration(evalIntervalMinutes)*time.Minute
}
