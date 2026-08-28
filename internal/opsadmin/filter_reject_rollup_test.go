package opsadmin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveFilterRejectMetricsURL_prefersDedicatedEnv(t *testing.T) {
	t.Setenv("FILTER_REJECT_METRICS_SCRAPE_URL", "http://127.0.0.1:9101/metrics")
	t.Setenv("TRACKER_METRICS_SCRAPE_URL", "http://127.0.0.1:9102/metrics")
	got := ResolveFilterRejectMetricsURL("http://fallback/metrics")
	if got != "http://127.0.0.1:9101/metrics" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterRejectRollupDeltas_holdoutNonZeroWhenSeeded(t *testing.T) {
	t.Parallel()

	prev := map[string]float64{"geo": 10, "edge_campaign_rl": 1}
	cur := map[string]float64{"geo": 15, "edge_campaign_rl": 2, "budget": 4}
	out := filterRejectRollupDeltas(prev, cur)
	assert.Equal(t, uint64(5), out["geo"])
	assert.Equal(t, uint64(1), out["edge_campaign_rl"])
	assert.Equal(t, uint64(4), out["budget"])
}

func TestMergeRejectCounterMaps_combinesTrackerAndEdge(t *testing.T) {
	t.Parallel()

	tracker := map[string]float64{"geo": 10}
	edge := map[string]float64{"edge_ip_blacklist": 3}
	out := mergeRejectCounterMaps(tracker, edge)
	assert.InDelta(t, 10, out["geo"], 0.001)
	assert.InDelta(t, 3, out["edge_ip_blacklist"], 0.001)
}
