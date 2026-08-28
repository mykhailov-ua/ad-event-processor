package automation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEvalIntervalMinutes_belowFloor_holdout(t *testing.T) {
	_, err := NormalizeEvalIntervalMinutes(5, 15)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 15")
}

func TestNormalizeEvalIntervalMinutes_allowedValues(t *testing.T) {
	got, err := NormalizeEvalIntervalMinutes(5, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestRuleDueForEval_5minWindow_holdout(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	last := now.Add(-4 * time.Minute)
	assert.False(t, RuleDueForEval(now, last, true, 5))
	assert.True(t, RuleDueForEval(now.Add(2*time.Minute), last, true, 5))
	assert.True(t, RuleDueForEval(now, time.Time{}, false, 5))
}
