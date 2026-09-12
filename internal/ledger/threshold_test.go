package ledger

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"

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

func TestWorkerInterval_defaultFiveSeconds_holdout(t *testing.T) {
	assert.Equal(t, 5*time.Second, WorkerInterval(nil))
	assert.Equal(t, 5*time.Second, WorkerInterval(&config.Config{MarginGuardIntervalSec: 0}))
	assert.Equal(t, 30*time.Second, WorkerInterval(&config.Config{MarginGuardIntervalSec: 30}))
}
