package opsadmin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrometheusMetrics_extractsInfraSeries(t *testing.T) {
	t.Parallel()
	body := `# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 1.610612736e+08
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 2.68435456e+08
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 42.5
# TYPE go_goroutines gauge
go_goroutines 128
`
	samples, err := parsePrometheusMetrics(strings.NewReader(body), "")
	require.NoError(t, err)
	names := make([]string, 0, len(samples))
	for _, s := range samples {
		names = append(names, s.Name)
	}
	assert.Contains(t, names, "go_memstats_heap_inuse_bytes")
	assert.Contains(t, names, "process_resident_memory_bytes")
	assert.Contains(t, names, "process_cpu_seconds_total")
	assert.Contains(t, names, "go_goroutines")
}

func TestOpsDashboardMetricCatalog_includesInfraMetrics(t *testing.T) {
	t.Parallel()
	require.Contains(t, OpsDashboardMetricCatalog, "process_resident_memory_bytes")
	require.Contains(t, OpsDashboardMetricCatalog, "go_memstats_heap_inuse_bytes")
}
