package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	samples := []forecastHourlySample{{hourOfDay: 10, impressions: 500}}
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	hours := enumerateActiveHours(start, start.Add(2*time.Hour), nil, "UTC")
	p50, p90 := impressionPercentiles(samples, hours, 5000)
	assert.Greater(t, p50, int64(0))
	assert.GreaterOrEqual(t, p90, p50)
}

func TestForecast_buildHourWeights_weighted(t *testing.T) {
	t.Parallel()
	w := buildHourWeights([]forecastHourlySample{{hourOfDay: 5, impressions: 100}, {hourOfDay: 6, impressions: 300}})
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
	w := buildHourWeights([]forecastHourlySample{{hourOfDay: 30, impressions: 10}})
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

func TestForecast_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "forecast", FileDomain("forecast_plan.go"))
}
