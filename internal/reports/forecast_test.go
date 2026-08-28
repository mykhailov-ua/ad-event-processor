package reports

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecast_evenPacingAdvisory_underfill(t *testing.T) {
	t.Parallel()
	adv := evenPacingAdvisory("EVEN", 10_000_000, 1_000, 5_000)
	require.NotNil(t, adv)
	assert.Equal(t, "PACING_UNDERFILL", adv.Code)
	assert.Equal(t, "ASAP", adv.SuggestedPacing)
}

func TestForecast_evenPacingAdvisory_noAdvisory_when_deliverable(t *testing.T) {
	t.Parallel()
	adv := evenPacingAdvisory("EVEN", 10_000_000, 10_000, 1_000)
	assert.Nil(t, adv)
}

func TestForecast_evenPacingAdvisory_skipped_for_ASAP(t *testing.T) {
	t.Parallel()
	adv := evenPacingAdvisory("ASAP", 10_000_000, 1_000, 5_000)
	assert.Nil(t, adv)
}

func TestForecast_enumerateActiveHours_daypart(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	hours := enumerateActiveHours(start, end, []int16{9, 10, 11}, "UTC")
	assert.Len(t, hours, 3)
}

func TestForecast_buildSpendCurve_EVEN_deterministic(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(4*time.Hour), nil, "UTC")
	curve := buildSpendCurve(hours, 4_000_000, "EVEN", 1_000)
	require.Len(t, curve, 4)
	for _, p := range curve {
		assert.Equal(t, int64(1_000_000), p.SpendMicro)
		assert.Equal(t, int64(1_000), p.Impressions)
	}
}

func TestForecast_impressionPercentiles_deterministic(t *testing.T) {
	t.Parallel()
	samples := []forecastHourlySample{{HourOfDay: 10, Impressions: 100}, {HourOfDay: 11, Impressions: 200}, {HourOfDay: 12, Impressions: 300}}
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(3*time.Hour), nil, "UTC")
	p50a, p90a := impressionPercentiles(samples, hours, 600)
	p50b, p90b := impressionPercentiles(samples, hours, 600)
	assert.Equal(t, p50a, p50b)
	assert.Equal(t, p90a, p90b)
}

func TestForecast_lowConfidence_threshold(t *testing.T) {
	t.Parallel()
	assert.True(t, int64(999) < forecastMinSampleImpressions)
	assert.False(t, int64(1000) < forecastMinSampleImpressions)
}

func TestForecast_normalizeForecastPacing(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "EVEN", normalizeForecastPacing("even"))
	assert.Equal(t, "ASAP", normalizeForecastPacing("ASAP"))
}

func TestForecast_buildHourWeights_emptyUsesUniform(t *testing.T) {
	t.Parallel()
	w := buildHourWeights(nil)
	for i := range w {
		assert.InDelta(t, 1.0/24.0, w[i], 1e-9)
	}
}

func TestForecast_projectFlightImpressions_zeroHours(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(0), projectFlightImpressions([24]float64{}, nil, 1000))
}

func TestForecast_impliedCPMMicro(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(5_000_000), impliedCPMMicro(5_000_000, 0))
	assert.Equal(t, int64(1000), impliedCPMMicro(10_000, 10))
}

func TestForecast_buildSpendCurve_ASAP(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(3*time.Hour), nil, "UTC")
	curve := buildSpendCurve(hours, 3_000_000, "ASAP", 1000)
	require.Len(t, curve, 3)
	assert.Greater(t, curve[0].SpendMicro, curve[2].SpendMicro)
}

func TestForecast_impressionPercentiles_withValues(t *testing.T) {
	t.Parallel()
	samples := []forecastHourlySample{{HourOfDay: 10, Impressions: 500}}
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(2*time.Hour), nil, "UTC")
	p50, p90 := impressionPercentiles(samples, hours, 5000)
	assert.Greater(t, p50, int64(0))
	assert.GreaterOrEqual(t, p90, p50)
}

func TestForecast_buildHourWeights_weighted(t *testing.T) {
	t.Parallel()
	w := buildHourWeights([]forecastHourlySample{{HourOfDay: 5, Impressions: 100}, {HourOfDay: 6, Impressions: 300}})
	assert.InDelta(t, 0.25, w[5], 1e-6)
	assert.InDelta(t, 0.75, w[6], 1e-6)
}

func TestForecast_projectFlightImpressions_weighted(t *testing.T) {
	t.Parallel()
	var weights [24]float64
	weights[12] = 1.0
	start := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	hours := []time.Time{start}
	got := projectFlightImpressions(weights, hours, 90*24*1000)
	assert.Greater(t, got, int64(0))
}

func TestForecast_buildHourWeights_zeroSumSamples(t *testing.T) {
	t.Parallel()
	w := buildHourWeights([]forecastHourlySample{{HourOfDay: 30, Impressions: 10}})
	for i := range w {
		assert.InDelta(t, 1.0/24.0, w[i], 1e-9)
	}
}

func TestForecast_enumerateActiveHours_invalidTimezone(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(2*time.Hour), nil, "Invalid/Zone")
	assert.Len(t, hours, 2)
}

func TestForecast_evenPacingAdvisory_nearThreshold(t *testing.T) {
	t.Parallel()
	adv := evenPacingAdvisory("EVEN", 10_000_000, 9_500, 1000)
	assert.Nil(t, adv)
}

func TestForecast_buildSpendCurve_emptyHours(t *testing.T) {
	t.Parallel()
	assert.Empty(t, buildSpendCurve(nil, 1_000_000, "EVEN", 1000))
}
