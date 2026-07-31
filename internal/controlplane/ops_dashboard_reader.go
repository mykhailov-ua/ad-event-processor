package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/controlplane/adminapi"
	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

const dashboardMetricsBucketSec = 300

func (r *opsReader) GetDashboardSummary(ctx context.Context) (adminapi.DashboardSummaryDTO, error) {
	if r == nil || r.svc == nil {
		return adminapi.DashboardSummaryDTO{}, fmt.Errorf("service not configured")
	}
	now := time.Now().UTC()
	snap, err := r.GetIncidentSnapshot(ctx)
	if err != nil {
		return adminapi.DashboardSummaryDTO{}, err
	}
	services := buildDashboardTopology(ctx, r.svc, snap)
	driftMax, rps, err := r.readDashboardLiveSignals(ctx, now)
	if err != nil {
		return adminapi.DashboardSummaryDTO{}, err
	}
	return adminapi.DashboardSummaryDTO{
		GeneratedAt:      now.Format(time.RFC3339),
		Services:         services,
		DriftMicroMax:    driftMax,
		DriftAlert:       driftMax > 0,
		RPSEstimate:      rps,
		OutboxPending:    snap.Outbox.Pending,
		EmergencyBreaker: snap.EmergencyBreaker,
	}, nil
}

func (r *opsReader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (adminapi.DashboardMetricsDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return adminapi.DashboardMetricsDTO{}, fmt.Errorf("postgres pool not configured")
	}
	if rangeHours <= 0 {
		rangeHours = 24
	}
	if rangeHours > 24 {
		rangeHours = 24
	}
	now := time.Now().UTC()
	since := now.Add(-time.Duration(rangeHours) * time.Hour)
	q := db.New(r.svc.GetPool())
	rows, err := q.ListOpsMetricSamplesDownsampled(ctx, db.ListOpsMetricSamplesDownsampledParams{
		Ts:      pgtype.Timestamptz{Time: since, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: metricName,
		Column4: float64(dashboardMetricsBucketSec),
	})
	if err != nil {
		return adminapi.DashboardMetricsDTO{}, err
	}
	points := make([]adminapi.DashboardMetricPoint, 0, len(rows))
	for _, row := range rows {
		ts, ok := metricSampleTime(row.Ts)
		if !ok {
			continue
		}
		points = append(points, adminapi.DashboardMetricPoint{
			Name:       row.Name,
			LabelsHash: row.LabelsHash,
			Timestamp:  ts.UTC().Format(time.RFC3339),
			Value:      row.Value,
		})
	}
	return adminapi.DashboardMetricsDTO{
		Range:       fmt.Sprintf("%dh", rangeHours),
		BucketSec:   dashboardMetricsBucketSec,
		Points:      points,
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

func buildDashboardTopology(ctx context.Context, svc *Service, snap adminapi.IncidentSnapshotDTO) []adminapi.DashboardServiceCard {
	cards := []adminapi.DashboardServiceCard{
		{ID: "management", Name: "Management", Status: "ok"},
		{ID: "tracker", Name: "Tracker", Status: "unknown"},
		{ID: "processor", Name: "Processor", Status: "unknown"},
	}
	if svc != nil && svc.GetPool() != nil {
		status := "ok"
		detail := ""
		if err := svc.GetPool().Ping(ctx); err != nil {
			status = "down"
			detail = err.Error()
		}
		cards = append(cards, adminapi.DashboardServiceCard{ID: "pg", Name: "Postgres", Status: status, Detail: detail})
	} else {
		cards = append(cards, adminapi.DashboardServiceCard{ID: "pg", Name: "Postgres", Status: "down"})
	}
	chStatus := "disabled"
	if svc != nil && svc.cfg != nil && svc.cfg.ClickHouseEnabled() {
		chStatus = "ok"
		if svc.CHQuery() == nil {
			chStatus = "down"
		}
	}
	cards = append(cards, adminapi.DashboardServiceCard{ID: "ch", Name: "ClickHouse", Status: chStatus})
	for _, shard := range snap.Shards {
		status := "ok"
		if !shard.PingOK {
			status = "down"
		}
		cards = append(cards, adminapi.DashboardServiceCard{
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

func (r *opsReader) readDashboardLiveSignals(ctx context.Context, now time.Time) (driftMax float64, rps float64, err error) {
	pool := r.svc.GetPool()
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
	first := rows[0]
	last := rows[len(rows)-1]
	if !first.Ts.Valid || !last.Ts.Valid {
		return driftMax, 0, nil
	}
	delta := last.Value - first.Value
	secs := last.Ts.Time.Sub(first.Ts.Time).Seconds()
	if secs > 0 && delta >= 0 {
		rps = delta / secs
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
