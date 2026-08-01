package controlplane

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"espx/internal/domain"
	db "espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type OutboxHealthSummary struct {
	Pending              int64   `json:"pending"`
	OldestPendingSeconds float64 `json:"oldest_pending_seconds"`
	LastProcessedEventID int64   `json:"last_processed_event_id"`
}

type ShardHealthStatus struct {
	ShardID             int     `json:"shard_id"`
	PingOK              bool    `json:"ping_ok"`
	PingError           string  `json:"ping_error,omitempty"`
	PingLatencyMs       float64 `json:"ping_latency_ms,omitempty"`
	ConfigVersion       *int64  `json:"config_version,omitempty"`
	ConfigVersionLag    int64   `json:"config_version_lag"`
	ConfigVersionSynced bool    `json:"config_version_synced"`
}

type ShardHealthReport struct {
	EmergencyBreaker string              `json:"emergency_breaker"`
	Outbox           OutboxHealthSummary `json:"outbox"`
	Shards           []ShardHealthStatus `json:"shards"`
}

func (s *Service) GetShardHealth(ctx context.Context) (ShardHealthReport, error) {
	var report ShardHealthReport
	report.Shards = make([]ShardHealthStatus, 0, len(s.rdbs))

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return report, fmt.Errorf("load system settings: %w", err)
	}
	report.EmergencyBreaker = settings["emergency_breaker"]

	outbox, err := s.outboxHealthSummary(ctx)
	if err != nil {
		return report, err
	}
	report.Outbox = outbox

	for shardID, rdb := range s.rdbs {
		status := probeShardHealth(ctx, shardID, rdb, outbox.LastProcessedEventID)
		report.Shards = append(report.Shards, status)
	}
	return report, nil
}

func (s *Service) outboxHealthSummary(ctx context.Context) (OutboxHealthSummary, error) {
	var summary OutboxHealthSummary
	if s.pool == nil {
		return summary, fmt.Errorf("postgres pool not configured")
	}
	err := s.pool.QueryRow(ctx, `
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

func probeShardHealth(ctx context.Context, shardID int, rdb redis.UniversalClient, lastProcessedEventID int64) ShardHealthStatus {
	status := ShardHealthStatus{ShardID: shardID}
	if rdb == nil {
		status.PingError = "redis client not configured"
		return status
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	start := time.Now()
	pingErr := rdb.Ping(pingCtx).Err()
	cancel()
	status.PingLatencyMs = float64(time.Since(start).Milliseconds())

	if pingErr != nil {
		status.PingError = pingErr.Error()
		return status
	}
	status.PingOK = true

	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	version, err := rdb.Get(versionCtx, redisConfigVersionKey).Int64()
	if err == redis.Nil {
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

type ShardMetrics struct {
	ShardID   int16
	CPUUsage  float64
	MemoryPct float64
	OpsPerSec int64
	LuaP99Ms  float64
}

type ShardMetricsProvider interface {
	GetMetrics(ctx context.Context, shardID int16, rdb redis.UniversalClient) (ShardMetrics, error)
}

type RealShardMetricsProvider struct{}

func (p *RealShardMetricsProvider) GetMetrics(ctx context.Context, shardID int16, rdb redis.UniversalClient) (ShardMetrics, error) {
	metrics := ShardMetrics{ShardID: shardID}

	memInfo, err := rdb.Info(ctx, "memory").Result()
	if err == nil {
		used := parseInfoInt64(memInfo, "used_memory")
		maxmem := parseInfoInt64(memInfo, "maxmemory")
		if maxmem > 0 {
			metrics.MemoryPct = (float64(used) / float64(maxmem)) * 100.0
		} else {
			metrics.MemoryPct = (float64(used) / (1024 * 1024 * 1024)) * 100.0
		}
	}

	statsInfo, err := rdb.Info(ctx, "stats").Result()
	if err == nil {
		metrics.OpsPerSec = parseInfoInt64(statsInfo, "instantaneous_ops_per_sec")
	}

	cpuInfo, err := rdb.Info(ctx, "cpu").Result()
	if err == nil {
		sys := parseInfoFloat64(cpuInfo, "used_cpu_sys")
		user := parseInfoFloat64(cpuInfo, "used_cpu_user")
		metrics.CPUUsage = (sys + user) * 10.0
		if metrics.CPUUsage > 100.0 {
			metrics.CPUUsage = 100.0
		}
	}

	return metrics, nil
}

type ShardAutoscaleConfig struct {
	Enabled        bool
	CPULimit       float64
	MemoryPctLimit float64
	OpsLimit       int64
	LuaP99Limit    float64
	SlotsToMigrate int16
}

func (s *Service) AutoscaleShards(ctx context.Context, provider ShardMetricsProvider, cfg ShardAutoscaleConfig) (int32, error) {
	if !cfg.Enabled || len(s.rdbs) <= 1 {
		return 0, nil
	}

	if provider == nil {
		provider = &RealShardMetricsProvider{}
	}

	if cfg.SlotsToMigrate <= 0 {
		cfg.SlotsToMigrate = 16
	}

	numShards := int16(len(s.rdbs))
	shardMetrics := make([]ShardMetrics, numShards)

	for i := int16(0); i < numShards; i++ {
		m, err := provider.GetMetrics(ctx, i, s.rdbs[i])
		if err != nil {
			continue
		}
		shardMetrics[i] = m
	}

	var maxShard int16 = -1
	var minShard int16 = -1
	var maxLoadScore float64 = -1.0
	var minLoadScore float64 = 1e18

	for i := int16(0); i < numShards; i++ {
		m := shardMetrics[i]
		memScore := m.MemoryPct / cfg.MemoryPctLimit
		opsScore := float64(m.OpsPerSec) / float64(cfg.OpsLimit)
		cpuScore := m.CPUUsage / cfg.CPULimit
		luaScore := m.LuaP99Ms / cfg.LuaP99Limit

		loadScore := memScore
		if opsScore > loadScore {
			loadScore = opsScore
		}
		if cpuScore > loadScore {
			loadScore = cpuScore
		}
		if luaScore > loadScore {
			loadScore = luaScore
		}

		isOverloaded := m.MemoryPct > cfg.MemoryPctLimit ||
			float64(m.OpsPerSec) > float64(cfg.OpsLimit) ||
			m.CPUUsage > cfg.CPULimit ||
			m.LuaP99Ms > cfg.LuaP99Limit

		if isOverloaded && loadScore > maxLoadScore {
			maxLoadScore = loadScore
			maxShard = i
		}

		if loadScore < minLoadScore {
			minLoadScore = loadScore
			minShard = i
		}
	}

	if maxShard == -1 || minShard == -1 || maxShard == minShard {
		return 0, nil
	}

	mapRepo := domain.NewSlotMapRepo(s.GetPool())
	activeVer, err := mapRepo.GetActiveVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get active slot map version: %w", err)
	}

	activeRows, err := mapRepo.ListVersion(ctx, activeVer)
	if err != nil {
		return 0, fmt.Errorf("failed to list active slot map rows: %w", err)
	}

	var selectedSlots []int16
	for _, row := range activeRows {
		if row.ShardID == maxShard && row.State == db.RedisSlotStateACTIVE {
			selectedSlots = append(selectedSlots, row.Slot)
			if int16(len(selectedSlots)) >= cfg.SlotsToMigrate {
				break
			}
		}
	}

	if len(selectedSlots) == 0 {
		return 0, nil
	}

	draftVer, err := s.CreateSlotMapVersion(ctx, uuid.Nil, &activeVer, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create draft slot map version: %w", err)
	}

	err = s.MarkSlotMapMigrating(ctx, uuid.Nil, draftVer, selectedSlots, minShard)
	if err != nil {
		return 0, fmt.Errorf("failed to mark slots migrating: %w", err)
	}

	err = s.EnsureSlotMigrationJobs(ctx, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to register slot migration jobs: %w", err)
	}

	err = s.CopyAllMigratingSlots(ctx, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to copy slot migration data: %w", err)
	}

	err = s.ActivateSlotMapVersion(ctx, uuid.Nil, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to activate new slot map version: %w", err)
	}

	_ = s.DrainMigratingSlots(ctx, draftVer)

	return draftVer, nil
}

func parseInfoInt64(info, key string) int64 {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				val, err := strconv.ParseInt(parts[1], 10, 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0
}

func parseInfoFloat64(info, key string) float64 {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				val, err := strconv.ParseFloat(parts[1], 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0.0
}
