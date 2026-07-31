package ledger

import (
	"testing"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestCostOverRevenueThresholdBps(t *testing.T) {
	policy := &Policy{CostOverRevenueThresholdBps: 250}
	cfg := &config.Config{MarginGuardDefaultThresholdBps: 500}
	assert.Equal(t, 250, CostOverRevenueThresholdBps(policy, cfg))
	assert.Equal(t, 500, CostOverRevenueThresholdBps(nil, cfg))
	assert.Equal(t, defaultCostOverRevenueThresholdBps, CostOverRevenueThresholdBps(nil, nil))
}

func TestCostOverRevenueLimitMicro(t *testing.T) {
	assert.Equal(t, int64(105_000), CostOverRevenueLimitMicro(100_000, 500))
}
