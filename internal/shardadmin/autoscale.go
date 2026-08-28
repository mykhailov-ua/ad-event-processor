package shardadmin

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

func AutoscaleShards(ctx context.Context, host OrchestratorHost, provider ShardMetricsProvider, cfg ShardAutoscaleConfig) (int32, error) {
	if !cfg.Enabled || len(host.RedisShards()) <= 1 {
		return 0, nil
	}

	if provider == nil {
		provider = &RealShardMetricsProvider{}
	}

	if cfg.SlotsToMigrate <= 0 {
		cfg.SlotsToMigrate = 16
	}

	numShards := int16(len(host.RedisShards()))
	shardMetrics := make([]ShardMetrics, numShards)

	for i := range numShards {
		m, err := provider.GetMetrics(ctx, i, host.RedisShards()[i])
		if err != nil {
			continue
		}
		shardMetrics[i] = m
	}

	var maxShard int16 = -1
	var minShard int16 = -1
	maxLoadScore := -1.0
	minLoadScore := 1e18

	for i := range numShards {
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

	mapRepo := domain.NewSlotMapRepo(host.Pool())
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

	draftVer, err := host.CreateSlotMapVersion(ctx, uuid.Nil, &activeVer, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create draft slot map version: %w", err)
	}

	err = host.MarkSlotMapMigrating(ctx, uuid.Nil, draftVer, selectedSlots, minShard)
	if err != nil {
		return 0, fmt.Errorf("failed to mark slots migrating: %w", err)
	}

	err = host.EnsureSlotMigrationJobs(ctx, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to register slot migration jobs: %w", err)
	}

	err = host.CopyAllMigratingSlots(ctx, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to copy slot migration data: %w", err)
	}

	err = host.ActivateSlotMapVersion(ctx, uuid.Nil, draftVer)
	if err != nil {
		return 0, fmt.Errorf("failed to activate new slot map version: %w", err)
	}

	if err := host.DrainMigratingSlots(ctx, draftVer); err != nil {
		return draftVer, fmt.Errorf("slot map activated but drain migrating slots failed: %w", err)
	}

	return draftVer, nil
}
