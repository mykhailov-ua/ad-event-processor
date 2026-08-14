package controlplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PaddedEma struct {
	Value float64
	_     [56]byte
}

type ShardOrchestrator struct {
	svc             *Service
	metricsProvider ShardMetricsProvider
	interval        time.Duration
	cooldown        time.Duration
	scaleThreshold  float64
	overloadLimit   time.Duration

	mu            sync.Mutex
	lastScaleTime time.Time
	overloadStart map[int16]time.Time
	shardEma      map[int16]*PaddedEma
	campaignEma   map[uuid.UUID]*PaddedEma
}

func NewShardOrchestrator(svc *Service, provider ShardMetricsProvider, interval time.Duration) *ShardOrchestrator {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &ShardOrchestrator{
		svc:             svc,
		metricsProvider: provider,
		interval:        interval,
		cooldown:        3600 * time.Second,
		scaleThreshold:  0.85,
		overloadLimit:   300 * time.Second,
		overloadStart:   make(map[int16]time.Time),
		shardEma:        make(map[int16]*PaddedEma),
		campaignEma:     make(map[uuid.UUID]*PaddedEma),
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
			o.tick(ctx)
		}
	}
}

func (o *ShardOrchestrator) tick(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	numShards := int16(len(o.svc.rdbs))
	if numShards <= 1 {
		return
	}

	alpha := 0.15
	var maxShard int16 = -1
	var maxEma float64 = -1.0

	for i := range numShards {
		m, err := o.metricsProvider.GetMetrics(ctx, i, o.svc.rdbs[i])
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
			ema = &PaddedEma{Value: rawScore}
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
	campaigns, err := o.svc.listActiveCampaignUUIDs(ctx)
	if err != nil {
		return err
	}

	sharder := domain.NewStaticSlotSharder(len(o.svc.rdbs))
	var bestCampaign uuid.UUID
	var maxCampaignLoad float64 = -1.0

	for _, id := range campaigns {
		if int16(sharder.GetShard(id)) == sourceShard {
			load := 0.5
			ema, ok := o.campaignEma[id]
			if !ok {
				ema = &PaddedEma{Value: load}
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
	var minEma float64 = 1e18
	for i := int16(0); i < int16(len(o.svc.rdbs)); i++ {
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

	routeRepo := domain.NewCampaignRoutingRepo(o.svc.GetPool())
	existing, _ := routeRepo.GetCampaignRouting(ctx, bestCampaign)
	routingEpoch := existing.RoutingEpoch + 1
	if routingEpoch <= 0 {
		var migrationGen int64
		if err := o.svc.GetPool().QueryRow(ctx, `SELECT migration_gen FROM campaigns WHERE id = $1`, domain.ToUUID(bestCampaign)).Scan(&migrationGen); err == nil {
			routingEpoch = migrationGen + 1
		} else {
			routingEpoch = 1
		}
	}

	homeSlot := domain.HomeSlotForCampaign(bestCampaign)
	if _, err := routeRepo.UpsertCampaignRouting(ctx, bestCampaign, homeSlot, targetShard, targetShard, targetShard, routingEpoch, 0.5, maxCampaignLoad); err != nil {
		return err
	}

	tx, err := o.svc.GetPool().Begin(ctx)
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

	srcRdb := o.svc.rdbs[sourceShard]
	dstRdb := o.svc.rdbs[targetShard]
	if err := domain.BumpMigrationFences(ctx, o.svc.GetPool(), srcRdb, []uuid.UUID{bestCampaign}); err != nil {
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
		o.svc.publishRoutingCutover(ctx, global.RoutingEpoch, global.ActiveVersion)
	}
	metrics.ElasticCampaignMigrationTotal.Inc()

	slog.Info("orchestrator: campaign migration completed successfully", "campaign", bestCampaign, "routing_epoch", routingEpoch)
	return nil
}

const (
	defaultDeadShardQuorum = 90 * time.Second
	trackerBreakerOpenPct  = 0.5
)

type ShardQuorumTracker struct {
	mu             sync.Mutex
	numShards      int
	quorum         time.Duration
	pingFailSince  []time.Time
	sentinelDown   []time.Time
	breakerOpen    []time.Time
	breakerPctFunc func(ctx context.Context, shard int) float64
}

func NewShardQuorumTracker(numShards int, quorum time.Duration) *ShardQuorumTracker {
	if quorum <= 0 {
		quorum = defaultDeadShardQuorum
	}
	if numShards <= 0 {
		numShards = 1
	}
	return &ShardQuorumTracker{
		numShards:     numShards,
		quorum:        quorum,
		pingFailSince: make([]time.Time, numShards),
		sentinelDown:  make([]time.Time, numShards),
		breakerOpen:   make([]time.Time, numShards),
	}
}

func (q *ShardQuorumTracker) SetBreakerPctFunc(fn func(ctx context.Context, shard int) float64) {
	q.mu.Lock()
	q.breakerPctFunc = fn
	q.mu.Unlock()
}

func (q *ShardQuorumTracker) ObserveShard(ctx context.Context, shard int, rdb redis.UniversalClient) {
	if q == nil || shard < 0 || shard >= q.numShards || rdb == nil {
		return
	}
	now := time.Now()

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	pingErr := rdb.Ping(pingCtx).Err()
	cancel()

	sentinelUp := pingErr == nil
	if pingErr == nil {
		infoCtx, infoCancel := context.WithTimeout(ctx, 2*time.Second)
		if info, err := rdb.Info(infoCtx, "replication").Result(); err != nil || info == "" {
			sentinelUp = false
		}
		infoCancel()
	}

	breakerPct := q.readBreakerPct(ctx, shard, rdb)

	q.mu.Lock()
	defer q.mu.Unlock()
	q.touch(&q.pingFailSince[shard], pingErr != nil, now)
	q.touch(&q.sentinelDown[shard], !sentinelUp, now)
	q.touch(&q.breakerOpen[shard], breakerPct >= trackerBreakerOpenPct, now)
}

func (q *ShardQuorumTracker) readBreakerPct(ctx context.Context, shard int, rdb redis.UniversalClient) float64 {
	if q.breakerPctFunc != nil {
		return q.breakerPctFunc(ctx, shard)
	}
	key := fmt.Sprintf("control:tracker_breaker_open_pct:%d", shard)
	v, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

func (q *ShardQuorumTracker) touch(slot *time.Time, active bool, now time.Time) {
	if active {
		if slot.IsZero() {
			*slot = now
		}
		return
	}
	*slot = time.Time{}
}

func (q *ShardQuorumTracker) DeadShardConfirmed(shard int) bool {
	if q == nil || shard < 0 || shard >= q.numShards {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	return heldFor(q.pingFailSince[shard], now, q.quorum) &&
		heldFor(q.sentinelDown[shard], now, q.quorum) &&
		heldFor(q.breakerOpen[shard], now, q.quorum)
}

func heldFor(since time.Time, now time.Time, d time.Duration) bool {
	return !since.IsZero() && now.Sub(since) >= d
}

type OutboxHealthSummary = adminapi.OutboxHealthSummary

type ShardHealthStatus = adminapi.ShardHealthStatus

type ShardHealthReport = adminapi.ShardHealthReport

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

	for i := range numShards {
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
