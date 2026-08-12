package ledger

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
)

const defaultCostOverRevenueThresholdBps = 500

func CostOverRevenueThresholdBps(policy *Policy, cfg *config.Config) int {
	if policy != nil && policy.CostOverRevenueThresholdBps > 0 {
		return policy.CostOverRevenueThresholdBps
	}
	if cfg != nil && cfg.MarginGuardDefaultThresholdBps > 0 {
		return cfg.MarginGuardDefaultThresholdBps
	}
	return defaultCostOverRevenueThresholdBps
}

func CostOverRevenueLimitMicro(advertiserSpendMicro int64, thresholdBps int) int64 {
	if advertiserSpendMicro <= 0 || thresholdBps < 0 {
		return 0
	}
	return int64(float64(advertiserSpendMicro) * (1.0 + float64(thresholdBps)/10000.0))
}

func WorkerInterval(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.MarginGuardIntervalSec <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(cfg.MarginGuardIntervalSec) * time.Second
}
