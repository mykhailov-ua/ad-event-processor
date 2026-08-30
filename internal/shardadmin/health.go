package shardadmin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const redisConfigVersionKey = "config:version"

// GetShardHealth: PG outbox pending count + per-shard Redis probe (2s timeout each).
func GetShardHealth(ctx context.Context, host HealthHost) (ShardHealthReport, error) {
	var report ShardHealthReport
	report.Shards = make([]ShardHealthStatus, 0, len(host.RedisShards()))

	settings, err := host.GetSettings(ctx)
	if err != nil {
		return report, fmt.Errorf("load system settings: %w", err)
	}
	report.EmergencyBreaker = settings["emergency_breaker"]

	outbox, err := QueryOutboxHealth(ctx, host.Pool())
	if err != nil {
		return report, err
	}
	report.Outbox = outbox

	for shardID, redisClient := range host.RedisShards() {
		status := probeShardHealth(ctx, shardID, redisClient, outbox.LastProcessedEventID)
		report.Shards = append(report.Shards, status)
	}
	return report, nil
}

func QueryOutboxHealth(ctx context.Context, pool *pgxpool.Pool) (OutboxHealthSummary, error) {
	var summary OutboxHealthSummary
	if pool == nil {
		return summary, fmt.Errorf("postgres pool not configured")
	}
	err := pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'PENDING')::bigint,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at) FILTER (WHERE status = 'PENDING'))), 0)::float8,
			COALESCE((SELECT MAX(id) FROM outbox_events WHERE status = 'PROCESSED'), 0)::bigint
		FROM outbox_events`,
	).Scan(&summary.Pending, &summary.OldestPendingSeconds, &summary.LastProcessedEventID)
	if err != nil {
		return summary, fmt.Errorf("query outbox health: %w", err)
	}
	return summary, nil
}

// probeShardHealth: config:version on shard vs last processed outbox id estimates control-plane fanout lag.
func probeShardHealth(ctx context.Context, shardID int, redisClient redis.UniversalClient, lastProcessedEventID int64) ShardHealthStatus {
	status := ShardHealthStatus{ShardID: shardID}
	if redisClient == nil {
		status.PingError = "redis client not configured"
		return status
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	start := time.Now()
	pingErr := redisClient.Ping(pingCtx).Err()
	cancel()
	status.PingLatencyMs = float64(time.Since(start).Milliseconds())

	if pingErr != nil {
		status.PingError = pingErr.Error()
		return status
	}
	status.PingOK = true

	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	version, err := redisClient.Get(versionCtx, redisConfigVersionKey).Int64()
	if errors.Is(err, redis.Nil) {
		if lastProcessedEventID > 0 {
			status.ConfigVersionLag = lastProcessedEventID
		}
		return status
	}
	if err != nil {
		status.PingOK = false
		status.PingError = fmt.Sprintf("read %s: %v", redisConfigVersionKey, err)
		return status
	}

	status.ConfigVersion = &version
	if version >= lastProcessedEventID {
		status.ConfigVersionSynced = true
		status.ConfigVersionLag = 0
	} else {
		status.ConfigVersionLag = lastProcessedEventID - version
	}
	return status
}
