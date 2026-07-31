package controlplane

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrometheusMetrics_extractsTrackedSeries(t *testing.T) {
	t.Parallel()
	body := `# HELP ad_http_requests_total HTTP requests
# TYPE ad_http_requests_total counter
ad_http_requests_total{method="GET",status="200"} 120
ad_http_requests_total{method="POST",status="500"} 3
# TYPE ad_recon_drift_micro gauge
ad_recon_drift_micro{campaign_id="a"} 10
ad_recon_drift_micro{campaign_id="b"} 25
# TYPE ad_management_outbox_pending_total gauge
ad_management_outbox_pending_total 4
# TYPE ad_tracker_redis_shard_healthy gauge
ad_tracker_redis_shard_healthy{shard="0"} 1
ignored_metric 99
`
	samples, err := parsePrometheusMetrics(strings.NewReader(body), "")
	require.NoError(t, err)
	require.NotEmpty(t, samples)

	var names []string
	for _, s := range samples {
		names = append(names, s.Name)
	}
	assert.Contains(t, names, "ad_http_requests_total")
	assert.Contains(t, names, "ad_recon_drift_micro_max")
	assert.Contains(t, names, "ad_management_outbox_pending_total")
	assert.Contains(t, names, "ad_tracker_redis_shard_healthy")

	var driftMax float64
	for _, s := range samples {
		if s.Name == "ad_recon_drift_micro_max" {
			driftMax = s.Value
		}
	}
	assert.Equal(t, 25.0, driftMax)
}
