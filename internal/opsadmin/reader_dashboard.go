package opsadmin

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain/db"

	"ad-event-processor/pkg/coldpath"

	"github.com/jackc/pgx/v5/pgtype"
)

const dashboardMetricsBucketSec = 300

const mlManualLabelsListLimit = 100

func (r *Reader) GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error) {
	if r == nil || r.pool() == nil {
		return DashboardSummaryDTO{}, fmt.Errorf("service not configured")
	}
	now := time.Now().UTC()
	snap, err := r.GetIncidentSnapshot(ctx)
	if err != nil {
		return DashboardSummaryDTO{}, err
	}
	services := buildDashboardTopology(ctx, r.deps, snap)
	driftMax, rps, err := r.readDashboardLiveSignals(ctx, now)
	if err != nil {
		return DashboardSummaryDTO{}, err
	}
	generatedAt := now.Format(time.RFC3339)
	return DashboardSummaryDTO{
		GeneratedAt:        generatedAt,
		GeneratedAtDisplay: coldpath.RFC3339Display(generatedAt),
		Services:           services,
		DriftMicroMax:      driftMax,
		DriftAlert:         driftMax > 0,
		RPSEstimate:        rps,
		OutboxPending:      snap.Outbox.Pending,
		EmergencyBreaker:   snap.EmergencyBreaker,
		Infra:              r.readInfraResourceSnapshot(ctx, now, rps),
	}, nil
}

func (r *Reader) GetStackHealthSnapshot(ctx context.Context) (StackHealthSnapshot, error) {
	if r == nil || r.pool() == nil {
		return StackHealthSnapshot{}, fmt.Errorf("service not configured")
	}
	return r.deps.BuildStackHealthSnapshot(ctx)
}

func (r *Reader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return DashboardMetricsDTO{}, fmt.Errorf("postgres pool not configured")
	}
	if rangeHours <= 0 {
		rangeHours = 24
	}
	if rangeHours > 24 {
		rangeHours = 24
	}
	now := time.Now().UTC()
	since := now.Add(-time.Duration(rangeHours) * time.Hour)
	q := db.New(r.pool())
	rows, err := q.ListOpsMetricSamplesDownsampled(ctx, db.ListOpsMetricSamplesDownsampledParams{
		Ts:      pgtype.Timestamptz{Time: since, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: metricName,
		Column4: float64(dashboardMetricsBucketSec),
	})
	if err != nil {
		return DashboardMetricsDTO{}, err
	}
	points := make([]DashboardMetricPoint, 0, len(rows))
	for _, row := range rows {
		ts, ok := metricSampleTime(row.Ts)
		if !ok {
			continue
		}
		points = append(points, DashboardMetricPoint{
			Name:       row.Name,
			LabelsHash: row.LabelsHash,
			Timestamp:  ts.UTC().Format(time.RFC3339),
			Value:      row.Value,
		})
	}
	return DashboardMetricsDTO{
		Range:       fmt.Sprintf("%dh", rangeHours),
		BucketSec:   dashboardMetricsBucketSec,
		Points:      points,
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

func buildDashboardTopology(ctx context.Context, deps ReaderDeps, snap IncidentSnapshotDTO) []DashboardServiceCard {
	cards := []DashboardServiceCard{
		{ID: "management", Name: "Management", Status: "ok"},
		{ID: "tracker", Name: "Tracker", Status: "unknown"},
		{ID: "processor", Name: "Processor", Status: "unknown"},
	}
	if deps.Pool != nil {
		status := "ok"
		detail := ""
		if err := deps.Pool.Ping(ctx); err != nil {
			status = "down"
			detail = err.Error()
		}
		cards = append(cards, DashboardServiceCard{ID: "postgres", Name: "Postgres", Status: status, Detail: detail})
	} else {
		cards = append(cards, DashboardServiceCard{ID: "postgres", Name: "Postgres", Status: "down"})
	}
	clickhouseStatus := "disabled"
	if deps.Config != nil && deps.Config.IsClickHouseEnabled() {
		clickhouseStatus = "ok"
		if deps.ClickHouseQuery == nil {
			clickhouseStatus = "down"
		}
	}
	cards = append(cards, DashboardServiceCard{ID: "clickhouse", Name: "ClickHouse", Status: clickhouseStatus})
	for _, shard := range snap.Shards {
		status := "ok"
		if !shard.PingOK {
			status = "down"
		}
		cards = append(cards, DashboardServiceCard{
			ID:     fmt.Sprintf("redis-%d", shard.ShardID),
			Name:   fmt.Sprintf("Redis %d", shard.ShardID),
			Status: status,
			Detail: shard.PingError,
		})
	}
	if snap.Outbox.Pending > 0 {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "degraded"
				cards[i].Detail = fmt.Sprintf("outbox_pending=%d", snap.Outbox.Pending)
			}
		}
	} else {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "ok"
			}
		}
	}
	return cards
}

func (r *Reader) readDashboardLiveSignals(ctx context.Context, now time.Time) (driftMax float64, rps float64, err error) {
	pool := r.pool()
	if pool == nil {
		return 0, 0, nil
	}
	q := db.New(pool)
	driftRow, derr := q.GetLatestOpsMetricSample(ctx, db.GetLatestOpsMetricSampleParams{
		Name:       "ad_recon_drift_micro_max",
		LabelsHash: "",
	})
	if derr == nil {
		driftMax = driftRow.Value
	}
	prevSince := now.Add(-2 * defaultOpsMetricScrapeInterval)
	rows, qerr := q.ListOpsMetricSamplesWindow(ctx, db.ListOpsMetricSamplesWindowParams{
		Ts:      pgtype.Timestamptz{Time: prevSince, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: "ad_http_requests_total",
	})
	if qerr != nil || len(rows) < 2 {
		return driftMax, 0, nil
	}
	window := make([]metricWindowRow, len(rows))
	for i, row := range rows {
		window[i] = metricWindowRow{Ts: row.Ts, Value: row.Value}
	}
	if rate, ok := counterRateFromLabeledSamples(metricSamplePointsFromWindow(window)); ok {
		rps = rate
	}
	return driftMax, rps, nil
}

func metricSampleTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case pgtype.Timestamptz:
		if t.Valid {
			return t.Time, true
		}
	}
	return time.Time{}, false
}
