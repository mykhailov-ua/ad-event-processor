package opsadmin

import (
	"context"
	"math"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

var opsScrapeMetricNames = map[string]struct{}{
	"ad_http_requests_total":          {},
	"ad_recon_drift_micro":            {},
	"ad_control_outbox_pending_total": {},
	"ad_tracker_redis_shard_healthy":  {},
	"go_memstats_heap_inuse_bytes":    {},
	"process_resident_memory_bytes":   {},
	"process_cpu_seconds_total":       {},
	"go_goroutines":                   {},
	"ad_gnet_active_connections":      {},
	"ad_worker_pool_reject_total":     {},
}

// OpsDashboardMetricCatalog is the allowlist for GET /api/v1/ops/dashboard/metrics name=.
var OpsDashboardMetricCatalog = []string{
	"ad_http_requests_total",
	"ad_control_outbox_pending_total",
	"ad_tracker_redis_shard_healthy",
	"process_resident_memory_bytes",
	"go_memstats_heap_inuse_bytes",
	"process_cpu_seconds_total",
	"go_goroutines",
	"ad_gnet_active_connections",
	"ad_worker_pool_reject_total",
}

func (r *Reader) readInfraResourceSnapshot(ctx context.Context, now time.Time, httpRPS float64) InfraResourceSnapshotDTO {
	out := InfraResourceSnapshotDTO{
		HTTPRPS:    httpRPS,
		ObservedAt: now.Format(time.RFC3339),
	}
	pool := r.pool()
	if pool == nil {
		out.ScrapeStale = true
		return out
	}
	q := db.New(pool)

	if v, ok := r.latestMetricGauge(ctx, q, "go_memstats_heap_inuse_bytes"); ok {
		out.HeapInuseBytes = int64(math.Round(v))
	}
	if v, ok := r.latestMetricGauge(ctx, q, "process_resident_memory_bytes"); ok {
		out.RSSBytes = int64(math.Round(v))
	}
	if v, ok := r.latestMetricGauge(ctx, q, "go_goroutines"); ok {
		out.Goroutines = int64(math.Round(v))
	}
	if v, ok := r.latestMetricGauge(ctx, q, "ad_gnet_active_connections"); ok {
		out.GnetConnections = int64(math.Round(v))
	}
	if v, ok := r.metricCounterRate(ctx, q, now, "process_cpu_seconds_total"); ok {
		out.CPUUtilizationPct = v * 100
	}

	out.ScrapeStale = out.HeapInuseBytes == 0 && out.RSSBytes == 0 && out.Goroutines == 0
	return out
}

func (r *Reader) latestMetricGauge(ctx context.Context, q *db.Queries, name string) (float64, bool) {
	row, err := q.GetLatestOpsMetricSample(ctx, db.GetLatestOpsMetricSampleParams{
		Name:       name,
		LabelsHash: "",
	})
	if err != nil {
		return 0, false
	}
	return row.Value, true
}

func (r *Reader) metricCounterRate(ctx context.Context, q *db.Queries, now time.Time, name string) (float64, bool) {
	prevSince := now.Add(-2 * defaultOpsMetricScrapeInterval)
	rows, err := q.ListOpsMetricSamplesWindow(ctx, db.ListOpsMetricSamplesWindowParams{
		Ts:      pgtype.Timestamptz{Time: prevSince, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: name,
	})
	if err != nil || len(rows) < 2 {
		return 0, false
	}
	window := make([]metricWindowRow, len(rows))
	for i, row := range rows {
		window[i] = metricWindowRow{Ts: row.Ts, Value: row.Value}
	}
	return counterRateFromLabeledSamples(metricSamplePointsFromWindow(window))
}
