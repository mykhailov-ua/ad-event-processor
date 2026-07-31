package controlplane

import (
	"context"
	"errors"
	"time"

	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func HistoricalSnapshotDay(now time.Time) time.Time {
	d := now.UTC().AddDate(0, 0, -1)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func LookupHistoricalDaily(
	ctx context.Context,
	pool *pgxpool.Pool,
	region int16,
	role, metric string,
	kind MetricKind,
	now time.Time,
) (*float64, error) {
	if pool == nil {
		return nil, nil
	}
	day := HistoricalSnapshotDay(now)
	q := db.New(pool)
	row, err := q.GetNodeMetricDailySnapshot(ctx, db.GetNodeMetricDailySnapshotParams{
		Day:        pgtype.Date{Time: day, Valid: true},
		RegionCode: region,
		Role:       role,
		Metric:     metric,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	raw, ok := historicalRawFromSnapshot(row, kind)
	if !ok {
		return nil, nil
	}
	return &raw, nil
}

func historicalRawFromSnapshot(row db.NodeMetricDailySnapshot, kind MetricKind) (float64, bool) {
	switch kind {
	case MetricLatency:
		if row.ValueP99.Valid {
			return row.ValueP99.Float64, true
		}
	case MetricUtilization, MetricRate, MetricCounter:
		if row.ValueMean.Valid {
			return row.ValueMean.Float64, true
		}
	default:
		if row.ValueMean.Valid {
			return row.ValueMean.Float64, true
		}
	}
	return 0, false
}
