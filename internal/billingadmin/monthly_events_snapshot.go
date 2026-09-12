package billingadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/licensingadmin"

	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

// RefreshDeploymentMonthlyEventsSnapshot reads billing.usage_meters, updates the in-process
// snapshot, and publishes to Redis for tracker replicas.
func RefreshDeploymentMonthlyEventsSnapshot(ctx context.Context, pool *pgxpool.Pool, redisClient redis.UniversalClient, quotaTimezone string) error {
	if pool == nil {
		return nil
	}
	used, err := licensingadmin.SumDeploymentAcceptedEventsMonth(ctx, pool, quotaTimezone)
	if err != nil {
		return err
	}
	syncedAt := time.Now().UTC()
	entitlements.UpdateDeploymentMonthlyEventsSnapshot(used)
	return entitlements.PublishDeploymentMonthlyEventsSnapshot(ctx, redisClient, used, syncedAt)
}
