package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestCollectors_DiskGateSeriesPresent(t *testing.T) {
	ControlScoreFallbackTotal.WithLabelValues("neighbor_median").Add(0)

	want := []string{
		"ad_disk_gate_append_wait_seconds",
		"ad_disk_gate_fsync_in_flight",
		"ad_disk_gate_shed_total",
		"ad_disk_gate_degraded",
		"ad_region_proxy_keygen_rate",
		"ad_region_proxy_keygen_queue_depth",
		"ad_region_proxy_keygen_lag_seconds",
		"ad_op_keypool_depth",
		"ad_region_proxy_ingress_shed_total",
		"ad_control_score_fallback_total",
	}

	fams, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	names := make(map[string]*dto.MetricFamily, len(fams))
	for _, fam := range fams {
		names[fam.GetName()] = fam
	}
	for _, name := range want {
		fam, ok := names[name]
		require.True(t, ok, "missing metric %s", name)
		require.NotEmpty(t, fam.GetHelp())
	}
}
