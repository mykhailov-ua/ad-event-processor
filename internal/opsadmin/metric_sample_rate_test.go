package opsadmin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggregateCounterByTimestamp_sumsLabeledSeriesPerScrape(t *testing.T) {
	t.Parallel()
	t0 := time.Unix(0, 0).UTC()
	t1 := t0.Add(15 * time.Second)
	points := []metricSamplePoint{
		{Ts: t0, Value: 100},
		{Ts: t0, Value: 50},
		{Ts: t1, Value: 130},
		{Ts: t1, Value: 70},
	}
	agg := aggregateCounterByTimestamp(points)
	require.Len(t, agg, 2)
	require.InDelta(t, 150, agg[0].Value, 1e-9)
	require.InDelta(t, 200, agg[1].Value, 1e-9)

	rate, ok := rateFromMonotonicCounterSeries(agg)
	require.True(t, ok)
	require.InDelta(t, 50.0/15.0, rate, 1e-9)
}

func TestCounterRateFromLabeledSamples_holdoutPlainEndpointsWrong(t *testing.T) {
	t.Parallel()
	t0 := time.Unix(0, 0).UTC()
	t1 := t0.Add(10 * time.Second)
	points := []metricSamplePoint{
		{Ts: t0, Value: 10},
		{Ts: t1, Value: 20},
		{Ts: t1, Value: 1000},
	}
	wrongDelta := points[len(points)-1].Value - points[0].Value
	wrongRate := wrongDelta / 10.0
	require.InDelta(t, 99, wrongRate, 1e-9)

	rate, ok := counterRateFromLabeledSamples(points)
	require.True(t, ok)
	require.InDelta(t, 101, rate, 1e-9)
}
