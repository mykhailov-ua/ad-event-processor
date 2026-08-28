package opsadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/shardadmin"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	stackHealthOutboxDegradedSeconds     = 30
	stackHealthOutboxCriticalSeconds     = 300
	stackHealthClickHouseDegradedSeconds = 300
	stackHealthClickHouseCriticalSeconds = 900
	stackHealthCostSyncDegradedSeconds   = 86400
	stackHealthCostSyncCriticalSeconds   = 172800
	stackHealthAutomationDegradedSeconds = 600
	stackHealthAutomationCriticalSeconds = 3600
)

type StackHealthDeps struct {
	Pool            *pgxpool.Pool
	LicenseState    func() (licensing.LicenseState, bool)
	ClickHouseLag   func(ctx context.Context) (time.Duration, error)
	ShardHealth     func(ctx context.Context) (shardadmin.ShardHealthReport, error)
	OutboxHealth    func(ctx context.Context, pool *pgxpool.Pool) (shardadmin.OutboxHealthSummary, error)
}

func BuildStackHealthSnapshot(ctx context.Context, deps StackHealthDeps) (StackHealthSnapshot, error) {
	if deps.Pool == nil && deps.LicenseState == nil {
		return StackHealthSnapshot{}, fmt.Errorf("stack health deps not configured")
	}

	snap := StackHealthSnapshot{
		LicenseState: string(licensing.StateExpired),
	}

	if deps.LicenseState != nil {
		if state, ok := deps.LicenseState(); ok {
			snap.LicenseState = string(state)
		}
	}

	if deps.ClickHouseLag != nil {
		if lag, err := deps.ClickHouseLag(ctx); err == nil {
			snap.ClickHouseLagSeconds = lag.Seconds()
		}
	}

	outboxFn := deps.OutboxHealth
	if outboxFn == nil {
		outboxFn = shardadmin.QueryOutboxHealth
	}
	if deps.Pool != nil {
		if outbox, err := outboxFn(ctx, deps.Pool); err == nil {
			snap.OutboxOldestPendingSeconds = outbox.OldestPendingSeconds
		}
	}

	if deps.ShardHealth != nil {
		if shardReport, err := deps.ShardHealth(ctx); err == nil {
			snap.RedisShardsTotal = len(shardReport.Shards)
			for _, shard := range shardReport.Shards {
				if shard.PingOK {
					snap.RedisShardsReachable++
				}
			}
			snap.RedisShardReachable = snap.RedisShardsTotal == 0 || snap.RedisShardsReachable > 0
		}
	}

	if deps.Pool != nil {
		var seconds float64
		err := deps.Pool.QueryRow(ctx, `
			SELECT COALESCE(EXTRACT(EPOCH FROM (NOW() - MAX(completed_at))), -1)::float8
			FROM cost_sync_runs
			WHERE status = 'success' AND completed_at IS NOT NULL`,
		).Scan(&seconds)
		if err == nil && seconds >= 0 {
			snap.CostSyncLastSuccessSeconds = &seconds
		}
	}

	if tick := automation.LastWorkerTick(); !tick.IsZero() {
		seconds := time.Since(tick).Seconds()
		snap.AutomationWorkerLastTickSeconds = &seconds
	}

	snap.Status = ComputeStackHealthStatus(snap)
	return snap, nil
}

func ComputeStackHealthStatus(snap StackHealthSnapshot) string {
	if snap.LicenseState == string(licensing.StateExpired) || snap.LicenseState == string(licensing.StateRevoked) {
		return "critical"
	}
	if snap.OutboxOldestPendingSeconds >= stackHealthOutboxCriticalSeconds {
		return "critical"
	}
	if snap.ClickHouseLagSeconds >= stackHealthClickHouseCriticalSeconds {
		return "critical"
	}
	if snap.RedisShardsTotal > 0 && snap.RedisShardsReachable == 0 {
		return "critical"
	}
	if snap.AutomationWorkerLastTickSeconds != nil && *snap.AutomationWorkerLastTickSeconds >= stackHealthAutomationCriticalSeconds {
		return "critical"
	}
	if snap.CostSyncLastSuccessSeconds != nil && *snap.CostSyncLastSuccessSeconds >= stackHealthCostSyncCriticalSeconds {
		return "critical"
	}

	degraded := false
	if snap.OutboxOldestPendingSeconds >= stackHealthOutboxDegradedSeconds {
		degraded = true
	}
	if snap.ClickHouseLagSeconds >= stackHealthClickHouseDegradedSeconds {
		degraded = true
	}
	if snap.RedisShardsTotal > 0 && snap.RedisShardsReachable < snap.RedisShardsTotal {
		degraded = true
	}
	if snap.AutomationWorkerLastTickSeconds != nil && *snap.AutomationWorkerLastTickSeconds >= stackHealthAutomationDegradedSeconds {
		degraded = true
	}
	if snap.CostSyncLastSuccessSeconds != nil && *snap.CostSyncLastSuccessSeconds >= stackHealthCostSyncDegradedSeconds {
		degraded = true
	}
	if snap.LicenseState == string(licensing.StateGrace) || snap.LicenseState == string(licensing.StateOfflineWarn) || snap.LicenseState == string(licensing.StateOfflineGrace) {
		degraded = true
	}
	if degraded {
		return "degraded"
	}
	return "ok"
}

func StackHealthSnapshotHasSecretMaterial(raw string) bool {
	lower := strings.ToLower(raw)
	needles := []string{
		"access_token",
		"refresh_token",
		"password",
		"postgres://",
		"redis://",
		"clickhouse://",
		"developer_token",
		"api_key",
	}
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}
