package controlplane

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/pgfailover"
	"ad-event-processor/internal/shardadmin"
)

var ErrPostgresGateRejected = errors.New("postgres gate rejected")

const postgresGateReserve = 1

// PostgresGate: high-priority admin/outbox slots vs low-priority reports; one conn reserved for probes.
type PostgresGate struct {
	sem      chan struct{}
	capacity int
	lowSlots chan struct{}
	inFlight atomic.Int32
}

func NewPostgresGate(maxConns int) *PostgresGate {
	capacity := maxConns - postgresGateReserve
	if capacity < 2 {
		capacity = 2
	}
	lowCap := capacity - 1
	if lowCap < 1 {
		lowCap = 1
	}
	return &PostgresGate{
		sem:      make(chan struct{}, capacity),
		capacity: capacity,
		lowSlots: make(chan struct{}, lowCap),
	}
}

func (g *PostgresGate) AcquireHigh(ctx context.Context) error {
	if g == nil {
		return nil
	}
	start := time.Now()
	select {
	case g.sem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			metrics.PostgresGateAcquireWaitSeconds.WithLabelValues("high").Observe(wait.Seconds())
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *PostgresGate) ReleaseHigh() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
}

func (g *PostgresGate) AcquireLow(ctx context.Context) error {
	if g == nil {
		return nil
	}
	select {
	case g.lowSlots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	default:
		metrics.PostgresGateRejectedTotal.WithLabelValues("low").Inc()
		return ErrPostgresGateRejected
	}

	start := time.Now()
	select {
	case g.sem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			metrics.PostgresGateAcquireWaitSeconds.WithLabelValues("low").Observe(wait.Seconds())
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		<-g.lowSlots
		return ctx.Err()
	}
}

func (g *PostgresGate) ReleaseLow() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
	<-g.lowSlots
}

func (g *PostgresGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

func (s *Service) withPostgresHigh(ctx context.Context, fn func(context.Context) error) error {
	if s == nil || s.postgresGate == nil {
		return fn(ctx)
	}
	if err := s.postgresGate.AcquireHigh(ctx); err != nil {
		return err
	}
	defer s.postgresGate.ReleaseHigh()
	return fn(ctx)
}

func (s *Service) withPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	if s == nil || s.postgresGate == nil {
		return fn(ctx)
	}
	if err := s.postgresGate.AcquireLow(ctx); err != nil {
		return err
	}
	defer s.postgresGate.ReleaseLow()
	return fn(ctx)
}

func (s *Service) WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error {
	return s.withPostgresHigh(ctx, fn)
}

func (s *Service) WithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return s.withPostgresLow(ctx, fn)
}

var ErrStalePgFencingEpoch = shardadmin.ErrStalePgFencingEpoch

var _ shardadmin.PostgresFailoverHost = (*Service)(nil)

func (s *Service) FailoverConfig() *config.Config {
	if s == nil {
		return nil
	}
	return s.cfg
}

func (s *Service) PgFencing() *pgfailover.FencingGate {
	if s == nil {
		return nil
	}
	return s.pgFencing
}

func (s *Service) SetPgFencing(g *pgfailover.FencingGate) {
	if s != nil {
		s.pgFencing = g
	}
}

func (s *Service) StartPostgresFailover(ctx context.Context) *shardadmin.PostgresFailoverRuntime {
	return shardadmin.StartPostgresFailover(ctx, s)
}

func (s *Service) requirePgFencing(ctx context.Context) error {
	return shardadmin.RequirePgFencing(ctx, s)
}

func (s *Service) AuditLedgerDuplicatesSinceFailover(ctx context.Context) (int, error) {
	return shardadmin.AuditLedgerDuplicatesSinceFailover(ctx, s)
}

func (s *Service) AutoscaleShards(ctx context.Context, provider shardadmin.ShardMetricsProvider, cfg shardadmin.ShardAutoscaleConfig) (int32, error) {
	return shardadmin.AutoscaleShards(ctx, s, provider, cfg)
}

func NewShardOrchestrator(svc *Service, provider shardadmin.ShardMetricsProvider, interval time.Duration) *shardadmin.ShardOrchestrator {
	return shardadmin.NewShardOrchestrator(svc, provider, interval)
}

func NewShard0CatchupWorker(svc *Service, redisOpts database.RedisShardOptions) *shardadmin.Shard0CatchupWorker {
	return shardadmin.NewShard0CatchupWorker(svc, redisOpts)
}

// TryReconnectShard0: lazy dial when shard 0 was nil at startup (degraded ingest path).
func (s *Service) TryReconnectShard0(ctx context.Context, opts database.RedisShardOptions) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	s.shard0Mu.Lock()
	defer s.shard0Mu.Unlock()

	if len(s.redisShards) == 0 || s.redisShards[0] != nil {
		return false
	}
	redisClient, err := database.ConnectRedisShard(ctx, s.cfg, 0, opts)
	if err != nil {
		return false
	}
	s.redisShards[0] = redisClient
	database.SetShard0ClientNilMetric(s.redisShards)
	return true
}

func (s *Service) RunShard0Catchup(ctx context.Context) error {
	return shardadmin.RunShard0Catchup(ctx, s)
}

func NewSlotMigrationOrchestrator(svc *Service, interval time.Duration) *shardadmin.SlotMigrationOrchestrator {
	return shardadmin.NewSlotMigrationOrchestrator(svc, interval)
}

// afterSlotMapActivated: bump routing_epoch in PG, reload StaticSlotSharder, publish TCP+broker cutover.
func (s *Service) afterSlotMapActivated(ctx context.Context, version int32) {
	routingEpoch := int64(0)
	if row, err := domain.NewCampaignRoutingRepo(s.GetPool()).BumpGlobalRoutingEpoch(ctx); err == nil {
		routingEpoch = row.RoutingEpoch
		version = row.ActiveVersion
	}
	if ss, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		_, _ = domain.LoadActiveSlotMap(ctx, s.GetPool(), ss, len(s.redisShards))
	}
	s.publishRoutingCutover(ctx, routingEpoch, version)
}
