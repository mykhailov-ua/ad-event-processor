package controlplane

import (
	"testing"
	"time"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_aggregateWindow_latency(t *testing.T) {
	t.Parallel()
	v, ok := aggregateWindow([]BucketPoint{{P99: 10}, {P99: 25}}, MetricLatency)
	require.True(t, ok)
	assert.Equal(t, 25.0, v)
}

func TestNode_aggregateWindow_utilization(t *testing.T) {
	t.Parallel()
	v, ok := aggregateWindow([]BucketPoint{{Mean: 0.2, SampleCount: 2}, {Mean: 0.4, SampleCount: 2}}, MetricUtilization)
	require.True(t, ok)
	assert.InDelta(t, 0.3, v, 1e-6)
}

func TestNode_ScorerConfigFrom_defaults(t *testing.T) {
	t.Parallel()
	cfg := ScorerConfigFrom(nil)
	assert.Equal(t, DefaultScorerConfig().WindowMin, cfg.WindowMin)
}

func TestNode_ScorerConfigFrom_overrides(t *testing.T) {
	t.Parallel()
	cfg := ScorerConfigFrom(&config.Config{NodeScoreWindowMin: 15, NodeScoreMinSamples: 9, NodeWarmupSec: 120})
	assert.Equal(t, 15, cfg.WindowMin)
	assert.Equal(t, 9, cfg.MinSamples)
	assert.Equal(t, 120, cfg.NodeWarmupSec)
}

func TestNode_ScoreNodes_batch(t *testing.T) {
	t.Parallel()
	cfg := DefaultScorerConfig()
	inputs := []NodeScoreInput{{
		Uptime: 30 * time.Minute,
		Kind:   MetricUtilization,
		Buckets: []BucketPoint{
			{Mean: 0.5, SampleCount: 10},
		},
		PreviousWeight: 0.5,
	}}
	out := ScoreNodes(inputs, cfg)
	require.Len(t, out, 1)
	assert.Greater(t, out[0].Weight, 0.0)
}

func TestNode_NormalizeMetricHealth_utilization(t *testing.T) {
	t.Parallel()
	defs := DefaultTrackerMetrics()
	require.NotEmpty(t, defs)
	score := NormalizeMetricHealth(0.45, defs[0])
	assert.InDelta(t, 0.5, score, 0.01)
}
