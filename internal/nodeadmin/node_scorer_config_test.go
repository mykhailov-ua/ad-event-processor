package nodeadmin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScoringWeightsJSON_badJSONFails(t *testing.T) {
	_, err := ParseScoringWeightsJSON(`{"tracker":`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse scoring weights json")
}

func TestParseScoringWeightsJSON_sumMustEqualOne(t *testing.T) {
	raw, err := json.Marshal(map[string]map[string]float64{
		RoleTracker: {
			MetricCPUUtil:              0.30,
			MetricRAMUtil:              0.15,
			MetricDiskFsyncP99MS:       0.15,
			MetricDiskGateWaitP99MS:    0.10,
			MetricIngressP99MS:         0.10,
			MetricFraudRejectRate:      0.10,
			MetricIVTRate:              0.08,
			MetricBudgetInvariantDrift: 0.07,
			MetricStreamLagBytes:       0.04,
		},
	})
	require.NoError(t, err)

	_, err = ParseScoringWeightsJSON(string(raw))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "want 1")
}

func TestParseScoringWeightsJSON_validTrackerWeights(t *testing.T) {
	raw, err := json.Marshal(map[string]map[string]float64{
		RoleTracker: {
			MetricCPUUtil:              0.20,
			MetricRAMUtil:              0.15,
			MetricDiskFsyncP99MS:       0.15,
			MetricDiskGateWaitP99MS:    0.10,
			MetricIngressP99MS:         0.10,
			MetricFraudRejectRate:      0.10,
			MetricIVTRate:              0.08,
			MetricBudgetInvariantDrift: 0.07,
			MetricStreamLagBytes:       0.05,
		},
	})
	require.NoError(t, err)

	got, err := ParseScoringWeightsJSON(string(raw))
	require.NoError(t, err)
	require.NotNil(t, got)

	defs := ApplyScoringWeights(DefaultTrackerMetrics(), got[RoleTracker])
	var sum float64
	for _, def := range defs {
		sum += def.Weight
	}
	assert.InDelta(t, 1.0, sum, 1e-9)
}

func TestRenormalizeMetricWeights_sumsToOne(t *testing.T) {
	in := map[string]float64{
		MetricCPUUtil: 0.2,
		MetricRAMUtil: 0.2,
		MetricIVTRate: 0.2,
	}
	out := RenormalizeMetricWeights(in)
	var sum float64
	for _, w := range out {
		sum += w
	}
	assert.InDelta(t, 1.0, sum, 1e-9)
}

func TestBuildScoringMetricDefsByRole_defaultsWhenEmpty(t *testing.T) {
	defs := BuildScoringMetricDefsByRole(nil)
	require.Len(t, defs[RoleTracker], len(DefaultTrackerMetrics()))
	var sum float64
	for _, def := range defs[RoleTracker] {
		sum += def.Weight
	}
	assert.InDelta(t, 1.0, sum, 1e-9)
}
