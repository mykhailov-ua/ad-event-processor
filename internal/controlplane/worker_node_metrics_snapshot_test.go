package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/nodeadmin"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeMetricDailyP99(t *testing.T) {
	t.Parallel()

	p99 := nodeadmin.NodeMetricDailyP99(float64(30))
	assert.Equal(t, pgtype.Float8{Float64: 30, Valid: true}, p99)

	pg := nodeadmin.NodeMetricDailyP99(pgtype.Float8{Float64: 42, Valid: true})
	assert.Equal(t, pgtype.Float8{Float64: 42, Valid: true}, pg)

	assert.False(t, nodeadmin.NodeMetricDailyP99("bad").Valid)
}

func TestNodeMetricsSnapshotWorker_RunOnce(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 2}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewNodeMetricsSnapshotWorker(svc)

	day := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	bucketTS := day.Add(12 * time.Hour)

	_, err := pool.Exec(ctx, `
		INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_p50, value_p99, value_mean, sample_count)
		VALUES
			('node-a', 2, 'tracker', $1, 'ingress_p99_ms', 10, 20, 15, 50),
			('node-b', 2, 'tracker', $1, 'ingress_p99_ms', 12, 30, 18, 60),
			('node-a', 2, 'tracker', $1, 'cpu_util', 0.4, 0.5, 0.45, 100)`,
		bucketTS)
	require.NoError(t, err)

	require.NoError(t, worker.RunOnce(ctx, day))

	q := db.New(pool)
	snap, err := q.GetNodeMetricDailySnapshot(ctx, db.GetNodeMetricDailySnapshotParams{
		Day:        pgtype.Date{Time: day, Valid: true},
		RegionCode: 2,
		Role:       "tracker",
		Metric:     "ingress_p99_ms",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(110), snap.SampleCount)
	assert.InDelta(t, 16.5, snap.ValueMean.Float64, 0.01)
	assert.InDelta(t, 30, snap.ValueP99.Float64, 0.01)

	cpuSnap, err := q.GetNodeMetricDailySnapshot(ctx, db.GetNodeMetricDailySnapshotParams{
		Day:        pgtype.Date{Time: day, Valid: true},
		RegionCode: 2,
		Role:       "tracker",
		Metric:     "cpu_util",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(100), cpuSnap.SampleCount)
	assert.InDelta(t, 0.45, cpuSnap.ValueMean.Float64, 0.01)

	require.NoError(t, worker.RunOnce(ctx, day))

	var rowCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM node_metric_daily_snapshots
		WHERE day = $1 AND region_code = 2 AND role = 'tracker'`, day).Scan(&rowCount))
	assert.Equal(t, 2, rowCount)
}

func TestLookupHistoricalDaily_ReadsPreviousDay(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	day := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	now := day.Add(24*time.Hour + 2*time.Hour)

	q := db.New(pool)
	require.NoError(t, q.UpsertNodeMetricDailySnapshot(ctx, db.UpsertNodeMetricDailySnapshotParams{
		Day:         pgtype.Date{Time: day, Valid: true},
		RegionCode:  1,
		Role:        "management",
		Metric:      "ingress_p99_ms",
		ValueP50:    pgtype.Float8{Float64: 10, Valid: true},
		ValueP99:    pgtype.Float8{Float64: 42, Valid: true},
		ValueMean:   pgtype.Float8{Float64: 20, Valid: true},
		SampleCount: 500,
	}))

	latency, err := LookupHistoricalDaily(ctx, pool, 1, "management", "ingress_p99_ms", MetricLatency, now)
	require.NoError(t, err)
	require.NotNil(t, latency)
	assert.InDelta(t, 42, *latency, 1e-9)

	util, err := LookupHistoricalDaily(ctx, pool, 1, "management", "ingress_p99_ms", MetricUtilization, now)
	require.NoError(t, err)
	require.NotNil(t, util)
	assert.InDelta(t, 20, *util, 1e-9)

	missing, err := LookupHistoricalDaily(ctx, pool, 1, "management", "missing_metric", MetricUtilization, now)
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestScoreNode_PhaseD_HistoricalDaily(t *testing.T) {
	cfg := DefaultScorerConfig()
	hist := 0.72
	res := ScoreNode(NodeScoreInput{
		Uptime:          20 * time.Minute,
		Kind:            MetricUtilization,
		Buckets:         []BucketPoint{{Mean: 0.1, SampleCount: 2}},
		HistoricalValue: &hist,
		PreviousWeight:  0.5,
		State:           NodeScoreState{EMAScore: 0.6},
	}, cfg)

	assert.Equal(t, ProvenanceHistoricalDaily, res.Provenance)
	assert.InDelta(t, 0.72, res.RawValue, 1e-9)
	assert.LessOrEqual(t, res.Weight, 0.5)
}

func TestNextSnapshotRunUTC(t *testing.T) {
	before := time.Date(2026, 7, 27, 0, 10, 0, 0, time.UTC)
	assert.Equal(t,
		time.Date(2026, 7, 27, 0, 15, 0, 0, time.UTC),
		nodeadmin.NextSnapshotRunUTC(before),
	)

	after := time.Date(2026, 7, 27, 1, 0, 0, 0, time.UTC)
	assert.Equal(t,
		time.Date(2026, 7, 28, 0, 15, 0, 0, time.UTC),
		nodeadmin.NextSnapshotRunUTC(after),
	)
}
