package shardadmin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

type paddedEma struct {
	Value float64
	_     [56]byte
}

type ShardOrchestrator struct {
	host            OrchestratorHost
	metricsProvider ShardMetricsProvider
	interval        time.Duration
	cooldown        time.Duration
	scaleThreshold  float64
	overloadLimit   time.Duration

	mu            sync.Mutex
	lastScaleTime time.Time
	overloadStart map[int16]time.Time
	shardEma      map[int16]*paddedEma
	campaignEma   map[uuid.UUID]*paddedEma
}

type OrchestratorTune struct {
	Cooldown       *time.Duration
	ScaleThreshold *float64
	OverloadLimit  *time.Duration
}

func NewShardOrchestrator(host OrchestratorHost, provider ShardMetricsProvider, interval time.Duration) *ShardOrchestrator {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &ShardOrchestrator{
		host:            host,
		metricsProvider: provider,
		interval:        interval,
		cooldown:        3600 * time.Second,
		scaleThreshold:  0.85,
		overloadLimit:   300 * time.Second,
		overloadStart:   make(map[int16]time.Time),
		shardEma:        make(map[int16]*paddedEma),
		campaignEma:     make(map[uuid.UUID]*paddedEma),
	}
}

func (o *ShardOrchestrator) Tune(t OrchestratorTune) {
	if t.Cooldown != nil {
		o.cooldown = *t.Cooldown
	}
	if t.ScaleThreshold != nil {
		o.scaleThreshold = *t.ScaleThreshold
	}
	if t.OverloadLimit != nil {
		o.overloadLimit = *t.OverloadLimit
	}
}

func (o *ShardOrchestrator) Start(ctx context.Context) {
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.Tick(ctx)
		}
	}
}

func (o *ShardOrchestrator) Tick(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	numShards := int16(len(o.host.RedisShards()))
	if numShards <= 1 {
		return
	}

	alpha := 0.15
	var maxShard int16 = -1
	maxEma := -1.0

	for i := range numShards {
		m, err := o.metricsProvider.GetMetrics(ctx, i, o.host.RedisShards()[i])
		if err != nil {
			slog.Warn("orchestrator: failed to get metrics", "shard", i, "error", err)
			continue
		}

		cpuScore := m.CPUUsage / 100.0
		memScore := m.MemoryPct / 100.0
		opsScore := 0.0
		if m.OpsPerSec > 0 {
			opsScore = float64(m.OpsPerSec) / 50000.0
		}

		rawScore := cpuScore
		if memScore > rawScore {
			rawScore = memScore
		}
		if opsScore > rawScore {
			rawScore = opsScore
		}

		ema, ok := o.shardEma[i]
		if !ok {
			ema = &paddedEma{Value: rawScore}
			o.shardEma[i] = ema
		} else {
			ema.Value = alpha*rawScore + (1.0-alpha)*ema.Value
		}

		if ema.Value > maxEma {
			maxEma = ema.Value
			maxShard = i
		}
	}

	if maxShard != -1 && maxEma >= o.scaleThreshold {
		start, ok := o.overloadStart[maxShard]
		if !ok {
			o.overloadStart[maxShard] = time.Now()
			slog.Info("orchestrator: shard capacity threshold exceeded", "shard", maxShard, "ema", maxEma)
		} else if time.Since(start) >= o.overloadLimit {
			if time.Since(o.lastScaleTime) >= o.cooldown {
				slog.Info("orchestrator: triggering scale-out migration", "shard", maxShard, "ema", maxEma)
				if err := o.migrateLoad(ctx, maxShard); err == nil {
					o.lastScaleTime = time.Now()
					delete(o.overloadStart, maxShard)
				} else {
					slog.Error("orchestrator: migration failed", "shard", maxShard, "error", err)
				}
			}
		}
	} else if maxShard != -1 {
		delete(o.overloadStart, maxShard)
	}
}

func (o *ShardOrchestrator) migrateLoad(ctx context.Context, sourceShard int16) error {
	campaigns, err := o.host.ListActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}

	sharder := domain.NewStaticSlotSharder(len(o.host.RedisShards()))
	var bestCampaign uuid.UUID
	maxCampaignLoad := -1.0

	for _, id := range campaigns {
		if int16(sharder.GetShard(id)) == sourceShard {
			load := 0.5
			ema, ok := o.campaignEma[id]
			if !ok {
				ema = &paddedEma{Value: load}
				o.campaignEma[id] = ema
			}
			if ema.Value > maxCampaignLoad {
				maxCampaignLoad = ema.Value
				bestCampaign = id
			}
		}
	}

	if bestCampaign == uuid.Nil {
		return fmt.Errorf("no campaign found on overloaded shard %d", sourceShard)
	}

	var targetShard int16 = -1
	minEma := 1e18
	for i := int16(0); i < int16(len(o.host.RedisShards())); i++ {
		if i == sourceShard {
			continue
		}
		ema, ok := o.shardEma[i]
		if ok && ema.Value < minEma {
			minEma = ema.Value
			targetShard = i
		}
	}

	if targetShard == -1 {
		return fmt.Errorf("no target shard found for migration")
	}

	slog.Info("orchestrator: initiating campaign migration", "campaign", bestCampaign, "source", sourceShard, "target", targetShard)

	routeRepo := domain.NewCampaignRoutingRepo(o.host.Pool())
	existing, _ := routeRepo.GetCampaignRouting(ctx, bestCampaign)
	routingEpoch := existing.RoutingEpoch + 1
	if routingEpoch <= 0 {
		var migrationGen int64
		if err := o.host.Pool().QueryRow(ctx, `SELECT migration_gen FROM campaigns WHERE id = $1`, domain.ToUUID(bestCampaign)).Scan(&migrationGen); err == nil {
			routingEpoch = migrationGen + 1
		} else {
			routingEpoch = 1
		}
	}

	homeSlot := domain.HomeSlotForCampaign(bestCampaign)
	if _, err := routeRepo.UpsertCampaignRouting(ctx, bestCampaign, homeSlot, targetShard, targetShard, targetShard, routingEpoch, 0.5, maxCampaignLoad); err != nil {
		return err
	}

	tx, err := o.host.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	row, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(bestCampaign))
	if err != nil {
		return err
	}
	_, err = q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     domain.ToUUID(bestCampaign),
		Status: row.Status,
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "UPDATE campaigns SET migration_gen = migration_gen + 1 WHERE id = $1", domain.ToUUID(bestCampaign))
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	srcRdb := o.host.RedisShards()[sourceShard]
	dstRdb := o.host.RedisShards()[targetShard]
	if err := domain.BumpMigrationFences(ctx, o.host.Pool(), srcRdb, []uuid.UUID{bestCampaign}); err != nil {
		return err
	}

	migrator := &domain.CampaignKeyMigrator{}
	if _, err := migrator.MigrateCampaignKeys(ctx, srcRdb, dstRdb, bestCampaign); err != nil {
		return err
	}
	if _, err := migrator.DrainCampaignKeys(ctx, srcRdb, bestCampaign); err != nil {
		return err
	}

	global, err := routeRepo.BumpGlobalRoutingEpoch(ctx)
	if err == nil {
		o.host.PublishRoutingCutover(ctx, global.RoutingEpoch, global.ActiveVersion)
	}
	metrics.ElasticCampaignMigrationTotal.Inc()

	slog.Info("orchestrator: campaign migration completed successfully", "campaign", bestCampaign, "routing_epoch", routingEpoch)
	return nil
}
