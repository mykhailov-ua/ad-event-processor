package entitlements

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	defaultMonthlyEventsStale        = 2 * time.Hour
	monthlyEventsDeploymentKey       = "entitlement:deployment"
	monthlyEventsUsedField           = "monthly_events_used"
	monthlyEventsSyncedAtField       = "monthly_events_synced_at"
	defaultMonthlyEventsPollInterval = 30 * time.Second
)

var (
	deploymentMonthlyEventsUsed     atomic.Uint64
	deploymentMonthlyEventsSyncedAt atomic.Int64
	monthlyEventsSyncActive         atomic.Bool
)

func UpdateDeploymentMonthlyEventsSnapshot(used uint64) {
	deploymentMonthlyEventsUsed.Store(used)
	deploymentMonthlyEventsSyncedAt.Store(time.Now().Unix())
}

func DeploymentMonthlyEventsSnapshot() (used uint64, syncedAt time.Time, ok bool) {
	ts := deploymentMonthlyEventsSyncedAt.Load()
	if ts == 0 {
		return 0, time.Time{}, false
	}
	return deploymentMonthlyEventsUsed.Load(), time.Unix(ts, 0), true
}

func PublishDeploymentMonthlyEventsSnapshot(ctx context.Context, redisClient redis.UniversalClient, used uint64, syncedAt time.Time) error {
	if redisClient == nil {
		return nil
	}
	return redisClient.HSet(ctx, monthlyEventsDeploymentKey, map[string]interface{}{
		monthlyEventsUsedField:     used,
		monthlyEventsSyncedAtField: syncedAt.Unix(),
	}).Err()
}

func ApplyDeploymentMonthlyEventsSnapshotFromRedis(ctx context.Context, redisClient redis.UniversalClient) bool {
	if redisClient == nil {
		return false
	}
	vals, err := redisClient.HMGet(ctx, monthlyEventsDeploymentKey, monthlyEventsUsedField, monthlyEventsSyncedAtField).Result()
	if err != nil || len(vals) != 2 {
		return false
	}
	usedStr, _ := vals[0].(string)
	syncedStr, _ := vals[1].(string)
	if usedStr == "" || syncedStr == "" {
		return false
	}
	used, err := strconv.ParseUint(usedStr, 10, 64)
	if err != nil {
		return false
	}
	syncedUnix, err := strconv.ParseInt(syncedStr, 10, 64)
	if err != nil || syncedUnix <= 0 {
		return false
	}
	deploymentMonthlyEventsUsed.Store(used)
	deploymentMonthlyEventsSyncedAt.Store(syncedUnix)
	return true
}

// StartMonthlyEventsSnapshotSync polls entitlement:deployment on tracker replicas.
// Control publishes via PublishDeploymentMonthlyEventsSnapshot after PG refresh.
func StartMonthlyEventsSnapshotSync(ctx context.Context, redisClient redis.UniversalClient, interval time.Duration) {
	if redisClient == nil || monthlyEventsSyncActive.Swap(true) {
		return
	}
	if interval <= 0 {
		interval = defaultMonthlyEventsPollInterval
	}
	applyCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ApplyDeploymentMonthlyEventsSnapshotFromRedis(applyCtx, redisClient)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollCtx, pollCancel := context.WithTimeout(ctx, 2*time.Second)
				ApplyDeploymentMonthlyEventsSnapshotFromRedis(pollCtx, redisClient)
				pollCancel()
			}
		}
	}()
}

func MonthlyEventsIngestAllowed(limits Limits, state LicenseState, licensed bool, now time.Time) bool {
	if !licensed || limitMonthlyEventsUnlimited(limits.MaxEventsPerMonth) {
		return true
	}
	if state == StateExpired || state == StateRevoked {
		return false
	}
	used, syncedAt, ok := DeploymentMonthlyEventsSnapshot()
	if !ok || now.Sub(syncedAt) > defaultMonthlyEventsStale {
		return false
	}
	return used < limits.MaxEventsPerMonth
}

func limitMonthlyEventsUnlimited(limit uint64) bool {
	return limit == 0 || limit >= 999999
}
