package management

import (
	"context"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpsDashboard_insertSample_queryAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	q := db.New(pool)
	require.NoError(t, q.InsertOpsMetricSample(ctx, db.InsertOpsMetricSampleParams{
		Name:       "ad_recon_drift_micro_max",
		LabelsHash: "",
		Ts:         pgtype.Timestamptz{Time: now, Valid: true},
		Value:      42,
	}))

	reader := &opsReader{svc: &Service{pool: pool}}
	metrics, err := reader.GetDashboardMetrics(ctx, 24, "ad_recon_drift_micro_max")
	require.NoError(t, err)
	require.NotEmpty(t, metrics.Points)
	assert.Equal(t, 42.0, metrics.Points[len(metrics.Points)-1].Value)

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()
	sharder := ingestion.NewStaticSlotSharder(1)
	svc := NewService(pool, []redis.UniversalClient{rdb}, sharder, &config.Config{})
	defer svc.Close()
	reader = &opsReader{svc: svc}
	summary, err := reader.GetDashboardSummary(ctx)
	require.NoError(t, err)
	assert.Equal(t, 42.0, summary.DriftMicroMax)
	assert.True(t, summary.DriftAlert)
}

func TestOpsDashboard_metricsQuery_10kSamples_under200ms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	q := db.New(pool)
	start := time.Now().UTC().Add(-24 * time.Hour)
	for i := 0; i < 10_000; i++ {
		ts := start.Add(time.Duration(i) * time.Second)
		require.NoError(t, q.InsertOpsMetricSample(ctx, db.InsertOpsMetricSampleParams{
			Name:       "ad_http_requests_total",
			LabelsHash: "",
			Ts:         pgtype.Timestamptz{Time: ts, Valid: true},
			Value:      float64(i),
		}))
	}

	reader := &opsReader{svc: &Service{pool: pool}}
	begin := time.Now()
	_, err := reader.GetDashboardMetrics(ctx, 24, "ad_http_requests_total")
	require.NoError(t, err)
	assert.Less(t, time.Since(begin), 200*time.Millisecond, "dashboard metrics query p99 budget")
}
