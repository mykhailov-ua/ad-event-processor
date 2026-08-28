package controlplane

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilterRejectCounters_holdoutExtractsReasonLabel(t *testing.T) {
	t.Parallel()

	body := `# HELP ad_filter_blocked_total blocked
# TYPE ad_filter_blocked_total counter
ad_filter_blocked_total{reason="geo"} 42
ad_filter_blocked_total{reason="budget"} 7
ad_filter_reject_country_total{reason="geo",country="US"} 3
`
	samples, err := parseFilterRejectCounters(strings.NewReader(body), "")
	require.NoError(t, err)
	require.Len(t, samples, 2)
	merged := mergeFilterRejectCounterSamples(samples)
	assert.InDelta(t, 42, merged["geo"], 0.001)
	assert.InDelta(t, 7, merged["budget"], 0.001)

	snap, err := parseFilterRejectMetrics(strings.NewReader(body), "")
	require.NoError(t, err)
	slices := mergeFilterRejectSliceSamples(snap.Slices)
	assert.InDelta(t, 3, slices["geo|US"], 0.001)
}

func TestFilterRejectCounterDelta_holdoutDetectsIncrement(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint64(5), filterRejectCounterDelta(10, 15))
	assert.Equal(t, uint64(0), filterRejectCounterDelta(10, 10))
	assert.Equal(t, uint64(3), filterRejectCounterDelta(10, 3))
}

func TestBuyerPortfolioEstimatedPacingDrift_heuristicOnly(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 65, pacingDriftPct(500, "ACTIVE"), 0.001)
	assert.Equal(t, float64(0), pacingDriftPct(500, "PAUSED"))
}
