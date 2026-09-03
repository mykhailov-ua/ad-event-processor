package opsadmin

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type metricSamplePoint struct {
	Ts    time.Time
	Value float64
}

type metricWindowRow struct {
	Ts    pgtype.Timestamptz
	Value float64
}

func metricSamplePointsFromWindow(rows []metricWindowRow) []metricSamplePoint {
	out := make([]metricSamplePoint, 0, len(rows))
	for _, row := range rows {
		if !row.Ts.Valid {
			continue
		}
		out = append(out, metricSamplePoint{Ts: row.Ts.Time.UTC(), Value: row.Value})
	}
	return out
}

func aggregateCounterByTimestamp(points []metricSamplePoint) []metricSamplePoint {
	if len(points) == 0 {
		return nil
	}
	agg := make([]metricSamplePoint, 0, len(points))
	cur := points[0]
	for i := 1; i < len(points); i++ {
		p := points[i]
		if p.Ts.Equal(cur.Ts) {
			cur.Value += p.Value
			continue
		}
		agg = append(agg, cur)
		cur = p
	}
	agg = append(agg, cur)
	return agg
}

func rateFromMonotonicCounterSeries(series []metricSamplePoint) (float64, bool) {
	if len(series) < 2 {
		return 0, false
	}
	first := series[0]
	last := series[len(series)-1]
	delta := last.Value - first.Value
	secs := last.Ts.Sub(first.Ts).Seconds()
	if secs <= 0 || delta < 0 {
		return 0, false
	}
	return delta / secs, true
}

func counterRateFromLabeledSamples(points []metricSamplePoint) (float64, bool) {
	return rateFromMonotonicCounterSeries(aggregateCounterByTimestamp(points))
}
