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

func TestFillDashboardSeriesPointEconomics_estimatesFromTraffic(t *testing.T) {
	t.Parallel()
	dayA := DashboardSeriesPointDTO{
		Label:       "2026-09-01",
		Clicks:      1000,
		Conversions: 40,
	}
	FillDashboardSeriesPointEconomicsGaps(&dayA)
	dayB := DashboardSeriesPointDTO{
		Label:       "2026-09-02",
		Clicks:      1000,
		Conversions: 40,
	}
	FillDashboardSeriesPointEconomicsGaps(&dayB)
	require.Greater(t, dayA.SpendMicro, int64(0))
	require.Greater(t, dayA.RevenueMicro, int64(0))
	require.Equal(t, dayA.RevenueMicro-dayA.SpendMicro, dayA.ProfitMicro)
	require.NotEqual(t, dayA.SpendMicro, dayB.SpendMicro)
	require.NotEqual(t, dayA.RevenueMicro, dayB.RevenueMicro)
}

func TestFillDashboardSeriesPointEconomics_revenueTracksConversionsNotClicks(t *testing.T) {
	t.Parallel()
	highConv := DashboardSeriesPointDTO{
		Label:       "2026-08-15",
		Clicks:      200_000,
		Conversions: 20_000,
	}
	FillDashboardSeriesPointEconomicsGaps(&highConv)
	lowConv := DashboardSeriesPointDTO{
		Label:       "2026-08-15",
		Clicks:      200_000,
		Conversions: 2_000,
	}
	FillDashboardSeriesPointEconomicsGaps(&lowConv)
	require.Less(t, lowConv.RevenueMicro, highConv.RevenueMicro)
	require.InDelta(t, float64(highConv.SpendMicro), float64(lowConv.SpendMicro), float64(highConv.SpendMicro)*0.08)
}

func TestFillDashboardSeriesPointEconomics_preservesReportedEconomics(t *testing.T) {
	t.Parallel()
	point := DashboardSeriesPointDTO{
		Clicks:       1000,
		SpendMicro:   5_000_000,
		RevenueMicro: 6_000_000,
	}
	FillDashboardSeriesPointEconomicsGaps(&point)
	require.Equal(t, int64(5_000_000), point.SpendMicro)
	require.Equal(t, int64(6_000_000), point.RevenueMicro)
	require.Equal(t, int64(1_000_000), point.ProfitMicro)
}
