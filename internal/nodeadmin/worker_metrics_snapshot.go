package nodeadmin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	snapshotRunHourUTC   = 0
	snapshotRunMinuteUTC = 15
)

type MetricsSnapshotWorker struct {
	host MetricsHost
	pool *pgxpool.Pool
}

func NewMetricsSnapshotWorker(host MetricsHost) *MetricsSnapshotWorker {
	return &MetricsSnapshotWorker{
		host: host,
		pool: host.Pool(),
	}
}

func NodeMetricDailyP99(v interface{}) pgtype.Float8 {
	return nodeMetricDailyP99(v)
}

func NextSnapshotRunUTC(now time.Time) time.Time {
	return nextSnapshotRunUTC(now)
}

func (w *MetricsSnapshotWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("node metrics snapshot worker starting", "run_at_utc", "00:15")

	now := time.Now().UTC()
	if !now.Before(todaySnapshotRunAt(now)) {
		if err := w.RunOnce(ctx, snapshotDayFor(now)); err != nil {
			slog.Error("node metrics snapshot catch-up failed", "err", err)
		}
	}

	for {
		now := time.Now().UTC()
		wait := time.Until(nextSnapshotRunUTC(now))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			day := snapshotDayFor(time.Now().UTC())
			if err := w.RunOnce(ctx, day); err != nil {
				slog.Error("node metrics snapshot failed", "day", day.Format("2006-01-02"), "err", err)
			}
		}
	}
}

func (w *MetricsSnapshotWorker) RunOnce(ctx context.Context, day time.Time) error {
	if w == nil || w.pool == nil {
		return nil
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	run := func(runCtx context.Context) error {
		return w.snapshotDay(runCtx, day)
	}
	if w.host != nil {
		return w.host.WithPostgresLow(ctx, run)
	}
	return run(ctx)
}

func (w *MetricsSnapshotWorker) snapshotDay(ctx context.Context, day time.Time) error {
	start := day
	end := day.Add(24 * time.Hour)
	q := db.New(w.pool)

	rows, err := q.AggregateNodeMetricBucketsForDay(ctx, db.AggregateNodeMetricBucketsForDayParams{
		BucketTs:   pgtype.Timestamptz{Time: start, Valid: true},
		BucketTs_2: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("aggregate node metric buckets day=%s: %w", day.Format("2006-01-02"), err)
	}

	dayParam := pgtype.Date{Time: day, Valid: true}
	for _, row := range rows {
		if err := q.UpsertNodeMetricDailySnapshot(ctx, db.UpsertNodeMetricDailySnapshotParams{
			Day:         dayParam,
			RegionCode:  row.RegionCode,
			Role:        row.Role,
			Metric:      row.Metric,
			ValueP50:    pgtype.Float8{Float64: row.ValueP50, Valid: true},
			ValueP99:    nodeMetricDailyP99(row.ValueP99),
			ValueMean:   pgtype.Float8{Float64: row.ValueMean, Valid: true},
			SampleCount: row.SampleCount,
		}); err != nil {
			return fmt.Errorf("upsert node metric daily snapshot day=%s metric=%s: %w",
				day.Format("2006-01-02"), row.Metric, err)
		}
	}

	slog.Info("node metric daily snapshots materialized",
		"day", day.Format("2006-01-02"),
		"rows", len(rows),
	)
	return nil
}

func snapshotDayFor(now time.Time) time.Time {
	d := now.UTC().AddDate(0, 0, -1)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func todaySnapshotRunAt(now time.Time) time.Time {
	n := now.UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), snapshotRunHourUTC, snapshotRunMinuteUTC, 0, 0, time.UTC)
}

func nextSnapshotRunUTC(now time.Time) time.Time {
	runAt := todaySnapshotRunAt(now)
	if !now.Before(runAt) {
		runAt = runAt.Add(24 * time.Hour)
	}
	return runAt
}

func nodeMetricDailyP99(v interface{}) pgtype.Float8 {
	switch x := v.(type) {
	case float64:
		return pgtype.Float8{Float64: x, Valid: true}
	case pgtype.Float8:
		return x
	default:
		return pgtype.Float8{}
	}
}
