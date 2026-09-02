package reports

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateChartRange_rejectsExcessiveWindow(t *testing.T) {
	t.Parallel()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(367 * 24 * time.Hour)
	err := ValidateChartRange(from, to)
	require.Error(t, err)
}

func TestValidateChartRange_acceptsYearWindow(t *testing.T) {
	t.Parallel()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(366 * 24 * time.Hour)
	require.NoError(t, ValidateChartRange(from, to))
}

func TestChartBucketWidth_hourlyForShortRange(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	require.Equal(t, time.Hour, chartBucketWidth(from, to))
}

func TestParseChartGranularity_defaultsToDay(t *testing.T) {
	t.Parallel()
	require.Equal(t, ChartGranularityDay, ParseChartGranularity(""))
	require.Equal(t, ChartGranularityDay, ParseChartGranularity("day"))
	require.Equal(t, ChartGranularityHour, ParseChartGranularity("hour"))
}

func TestValidateChartGranularityRange_rejectsLongHourlyWindow(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(32 * 24 * time.Hour)
	err := ValidateChartGranularityRange(ChartGranularityHour, from, to)
	require.Error(t, err)
}
