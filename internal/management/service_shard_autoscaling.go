package management

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"espx/internal/ingestion"
	"espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

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

	mapRepo := ingestion.NewSlotMapRepo(s.GetPool())
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
