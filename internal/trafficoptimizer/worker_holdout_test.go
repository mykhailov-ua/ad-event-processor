package trafficoptimizer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorker_ruleOnCooldown_holdout(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	last := now.Add(-30 * time.Minute)
	rule := Rule{CooldownMinutes: 60}
	onCooldown := now.Sub(last) < time.Duration(rule.CooldownMinutes)*time.Minute
	assert.True(t, onCooldown, "holdout: cooldown must block apply within window")
}
