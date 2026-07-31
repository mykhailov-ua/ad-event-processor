package controlplane

import (
	"context"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeMetricsWorker_FlushWindowAndTTL(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "metrics-node-1",
		NodeRole:           "management",
	}
	svc := newBareService(t, pool, nil, cfg)
	worker := NewNodeMetricsWorker(svc)

	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		worker.Record("ingress_p99_ms", float64(i))
	}
	require.NoError(t, worker.Flush(ctx, now))

	q := db.New(pool)
	rows, err := q.ListNodeMetricBucketsWindow(ctx, db.ListNodeMetricBucketsWindowParams{
		RegionCode: 1,
		Role:       "management",
		BucketTs:   pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true},
		BucketTs_2: pgtype.Timestamptz{Time: now.Add(time.Minute), Valid: true},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(100), rows[0].SampleCount)
	assert.InDelta(t, 49.5, rows[0].ValueMean.Float64, 0.1)
	assert.InDelta(t, 98.01, rows[0].ValueP99.Float64, 0.5)

	oldTS := now.Add(-25 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_mean, sample_count)
		VALUES ($1, 1, 'management', $2, 'stale_metric', 1.0, 1)`,
		worker.nodeID, oldTS)
	require.NoError(t, err)

	require.NoError(t, worker.Flush(ctx, now))

	var staleCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM node_metric_buckets
		WHERE metric = 'stale_metric' AND bucket_ts = $1`, oldTS).Scan(&staleCount))
	assert.Equal(t, 0, staleCount)

	var freshCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM node_metric_buckets
		WHERE metric = 'ingress_p99_ms'`).Scan(&freshCount))
	assert.Equal(t, 1, freshCount)
}

func TestAggregateSamples_percentiles(t *testing.T) {
	p50, p99, mean, count := aggregateSamples([]float64{1, 2, 3, 4, 100})
	assert.Equal(t, int64(5), count)
	assert.InDelta(t, 22, mean, 0.01)
	assert.InDelta(t, 3, p50, 0.01)
	assert.Greater(t, p99, 90.0)
}
