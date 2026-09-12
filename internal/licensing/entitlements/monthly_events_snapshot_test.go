package entitlements

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestMonthlyEventsIngestAllowed_unlimited(t *testing.T) {
	require.True(t, MonthlyEventsIngestAllowed(Limits{}, StateActive, true, time.Now()))
}

func TestMonthlyEventsIngestAllowed_staleSnapshotBlocks(t *testing.T) {
	limits := Limits{MaxEventsPerMonth: 100}
	require.False(t, MonthlyEventsIngestAllowed(limits, StateActive, true, time.Now()))
}

func TestMonthlyEventsIngestAllowed_underCap(t *testing.T) {
	UpdateDeploymentMonthlyEventsSnapshot(50)
	t.Cleanup(func() { deploymentMonthlyEventsSyncedAt.Store(0) })
	limits := Limits{MaxEventsPerMonth: 100}
	require.True(t, MonthlyEventsIngestAllowed(limits, StateActive, true, time.Now()))
}

func TestMonthlyEventsIngestAllowed_atCap(t *testing.T) {
	UpdateDeploymentMonthlyEventsSnapshot(100)
	t.Cleanup(func() { deploymentMonthlyEventsSyncedAt.Store(0) })
	limits := Limits{MaxEventsPerMonth: 100}
	require.False(t, MonthlyEventsIngestAllowed(limits, StateActive, true, time.Now()))
}

func TestDeploymentMonthlyEventsSnapshot_redisRoundTrip(t *testing.T) {
	deploymentMonthlyEventsSyncedAt.Store(0)
	t.Cleanup(func() { deploymentMonthlyEventsSyncedAt.Store(0) })

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	syncedAt := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, PublishDeploymentMonthlyEventsSnapshot(ctx, rdb, 42, syncedAt))
	require.True(t, ApplyDeploymentMonthlyEventsSnapshotFromRedis(ctx, rdb))

	used, gotSyncedAt, ok := DeploymentMonthlyEventsSnapshot()
	require.True(t, ok)
	require.Equal(t, uint64(42), used)
	require.Equal(t, syncedAt.Unix(), gotSyncedAt.Unix())
}
