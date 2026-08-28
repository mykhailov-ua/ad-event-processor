package controlplane

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/opsadmin"
)

const (
	stackHealthOutboxDegradedSeconds       = 30
	stackHealthOutboxCriticalSeconds       = 300
	stackHealthClickHouseDegradedSeconds   = 300
	stackHealthClickHouseCriticalSeconds   = 900
	stackHealthCostSyncDegradedSeconds     = 86400
	stackHealthCostSyncCriticalSeconds     = 172800
	stackHealthAutomationDegradedSeconds   = 600
	stackHealthAutomationCriticalSeconds   = 3600
)

func (s *Service) BuildStackHealthSnapshot(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
	if s == nil {
		return opsadmin.StackHealthSnapshot{}, fmt.Errorf("service not configured")
	}

	snap := opsadmin.StackHealthSnapshot{
		LicenseState: string(licensing.StateExpired),
	}

	state, ok := licenseWatcherState()
	if ok {
		snap.LicenseState = string(state)
	}

	if lag, err := s.clickHouseIngestionLag(ctx); err == nil {
		snap.ClickHouseLagSeconds = lag.Seconds()
	}

	if outbox, err := s.outboxHealthSummary(ctx); err == nil {
		snap.OutboxOldestPendingSeconds = outbox.OldestPendingSeconds
	}

	shardReport, err := s.GetShardHealth(ctx)
	if err == nil {
		snap.RedisShardsTotal = len(shardReport.Shards)
		for _, shard := range shardReport.Shards {
			if shard.PingOK {
				snap.RedisShardsReachable++
			}
		}
		snap.RedisShardReachable = snap.RedisShardsTotal == 0 || snap.RedisShardsReachable > 0
	}

	if s.pool != nil {
		var seconds float64
		err := s.pool.QueryRow(ctx, `
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

	snap.Status = computeStackHealthStatus(snap)
	return snap, nil
}

func computeStackHealthStatus(snap StackHealthSnapshot) string {
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

func stackHealthSnapshotHasSecretMaterial(raw string) bool {
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
