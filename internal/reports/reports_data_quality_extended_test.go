package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildReportLagMap_includesCatalogKeys(t *testing.T) {
	t.Parallel()
	lag := buildReportLagMap(DataFreshnessDTO{CHLagSeconds: 42, Stale: true})
	require.NotEmpty(t, lag)
	require.Equal(t, int32(42), lag["customer-fraud-by-type"])
	require.Equal(t, int32(42), lag["data-quality"])
}

func TestEstimateTelemetryMissingRate_emptyRows(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0.0, estimateTelemetryMissingRate(nil))
}
