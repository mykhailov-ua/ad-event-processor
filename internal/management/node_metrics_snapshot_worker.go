package management

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const snapshotRunHourUTC = 0
const snapshotRunMinuteUTC = 15

type NodeMetricsSnapshotWorker struct {
	svc  *Service
	pool *pgxpool.Pool
}

func NewNodeMetricsSnapshotWorker(svc *Service) *NodeMetricsSnapshotWorker {
	return &NodeMetricsSnapshotWorker{
		svc:  svc,
		pool: svc.GetPool(),
	}
}

func (w *NodeMetricsSnapshotWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("node metrics snapshot worker starting", "run_at_utc", "00:15")

	now := time.Now().UTC()
	if !now.Before(todaySnapshotRunAt(now)) {
		if err := w.RunOnce(ctx, snapshotDayFor(now)); err != nil {
			slog.Error("node metrics snapshot catch-up failed", "error", err)
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
				slog.Error("node metrics snapshot failed", "day", day.Format("2006-01-02"), "error", err)
			}
		}
	}
}

func (w *NodeMetricsSnapshotWorker) RunOnce(ctx context.Context, day time.Time) error {
	if w == nil || w.pool == nil {
		return nil
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	run := func(runCtx context.Context) error {
		return w.snapshotDay(runCtx, day)
	}
	if w.svc != nil {
		return w.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (w *NodeMetricsSnapshotWorker) snapshotDay(ctx context.Context, day time.Time) error {
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
		p99 := aggregateFloat(row.ValueP99)
		if err := q.UpsertNodeMetricDailySnapshot(ctx, db.UpsertNodeMetricDailySnapshotParams{
			Day:         dayParam,
			RegionCode:  row.RegionCode,
			Role:        row.Role,
			Metric:      row.Metric,
			ValueP50:    pgtype.Float8{Float64: row.ValueP50, Valid: true},
			ValueP99:    pgtype.Float8{Float64: p99, Valid: true},
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

func aggregateFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	default:
		return 0
	}
}
