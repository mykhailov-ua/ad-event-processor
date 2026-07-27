package management

import (
	"context"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	db "espx/internal/ingestion/sqlc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeCapacityScorer_TickWritesScores(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{
		MultiRegionEnabled:  true,
		RegionCode:          3,
		NodeScoreWindowMin:  5,
		NodeScoreMinSamples: 30,
		UDPSyncIntervalMs:   10000,
	}
	svc := newBareService(t, pool, nil, cfg)
	scorer := NewNodeCapacityScorer(svc)

	now := time.Date(2026, 7, 27, 14, 0, 0, 0, time.UTC)
	windowStart := now.Add(-6 * time.Minute)

	healthy := map[string]float64{
		MetricCPUUtil:              0.40,
		MetricRAMUtil:              0.35,
		MetricDiskFsyncP99MS:       12,
		MetricDiskGateWaitP99MS:    6,
		MetricIngressP99MS:         55,
		MetricFraudRejectRate:      0.01,
		MetricIVTRate:              0.01,
		MetricBudgetInvariantDrift: 0,
		MetricStreamLagBytes:       50_000,
	}
	unhealthy := map[string]float64{
		MetricCPUUtil:              0.85,
		MetricRAMUtil:              0.80,
		MetricDiskFsyncP99MS:       45,
		MetricDiskGateWaitP99MS:    25,
		MetricIngressP99MS:         90,
		MetricFraudRejectRate:      0.04,
		MetricIVTRate:              0.025,
		MetricBudgetInvariantDrift: 0.5,
		MetricStreamLagBytes:       800_000,
	}
	for i := 0; i < 40; i++ {
		ts := windowStart.Add(time.Duration(i) * 10 * time.Second)
		for metric, mean := range healthy {
			_, err := pool.Exec(ctx, `
				INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_mean, sample_count)
				VALUES ('tracker-a', 3, 'tracker', $1, $2, $3, 2)`, ts, metric, mean)
			require.NoError(t, err)
		}
		for metric, mean := range unhealthy {
			_, err := pool.Exec(ctx, `
				INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_mean, sample_count)
				VALUES ('tracker-b', 3, 'tracker', $1, $2, $3, 2)`, ts, metric, mean)
			require.NoError(t, err)
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_mean, sample_count)
		VALUES ('tracker-b', 3, 'tracker', $1, $2, 1, 1)`,
		now.Add(-time.Second), MetricDiskDegraded)
	require.NoError(t, err)

	require.NoError(t, scorer.Tick(ctx, now))

	q := db.New(pool)
	rows, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
		RegionCode: 3,
		Role:       RoleTracker,
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byNode := make(map[string]db.NodeCapacityScore, len(rows))
	for _, row := range rows {
		byNode[row.NodeID] = row
	}
	assert.Greater(t, byNode["tracker-a"].Score, byNode["tracker-b"].Score)
	assert.Equal(t, 0.0, byNode["tracker-b"].Weight)

	var weightSum float64
	for _, row := range rows {
		weightSum += row.Weight
	}
	assert.InDelta(t, 1.0, weightSum, 1e-6)
	assert.Greater(t, rows[0].EpochID, int64(0))
}

func TestNodeCapacityScorer_noCrossRegionMixing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1}
	svc := newBareService(t, pool, nil, cfg)
	scorer := NewNodeCapacityScorer(svc)
	now := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	ts := now.Add(-2 * time.Minute)

	_, err := pool.Exec(ctx, `
		INSERT INTO node_metric_buckets (node_id, region_code, role, bucket_ts, metric, value_mean, sample_count)
		VALUES
			('eu-node', 1, 'tracker', $1, $2, 0.2, 50),
			('us-node', 2, 'tracker', $1, $2, 0.9, 50)`,
		ts, MetricCPUUtil)
	require.NoError(t, err)

	require.NoError(t, scorer.Tick(ctx, now))

	q := db.New(pool)
	rows, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
		RegionCode: 1,
		Role:       RoleTracker,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "eu-node", rows[0].NodeID)
}
