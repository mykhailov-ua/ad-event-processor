package controlplane

import (
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/shardadmin"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	_ shardadmin.Host             = (*Service)(nil)
	_ shardadmin.HealthHost       = (*Service)(nil)
	_ shardadmin.OrchestratorHost = (*Service)(nil)
	_ shardadmin.LeaseHost        = (*Service)(nil)
	_ shardadmin.CatchupHost      = (*Service)(nil)
)

func (s *Service) AlertSlotMapMigrating(ctx context.Context, version int32, slots []int16, targetShard int16) {
	if s != nil && s.alerter != nil {
		s.alerter.AlertSlotMapMigrating(ctx, version, slots, targetShard)
	}
}

func (s *Service) AlertSlotMigrationError(ctx context.Context, stage string, err error) {
	if s != nil && s.alerter != nil {
		s.alerter.AlertSlotMigrationError(ctx, stage, err)
	}
}

func (s *Service) ListActiveCampaignUUIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := db.New(s.GetPool()).ListCampaignIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if !row.Valid {
			continue
		}
		out = append(out, uuid.UUID(row.Bytes))
	}
	return out, nil
}

func (s *Service) SlotMigrationDualWriteEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.SlotMigrationDualWriteEnabled
}

func (s *Service) DualWriteConfig() domain.SlotMigrationDualWriteConfig {
	cfg := domain.SlotMigrationDualWriteConfig{
		Enabled:      s.SlotMigrationDualWriteEnabled(),
		LagEpsilon:   0,
		LagThreshold: 1000,
	}
	if s == nil || s.cfg == nil {
		return cfg
	}
	cfg.LagEpsilon = s.cfg.SlotMigrationLagEpsilon
	cfg.LagThreshold = s.cfg.SlotMigrationLagThreshold
	if cfg.LagThreshold <= 0 {
		cfg.LagThreshold = 1000
	}
	return cfg
}

func (s *Service) MigrationFenceEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.MigrationFenceEnabled
}

func (s *Service) AfterSlotMapActivated(ctx context.Context, version int32) {
	s.afterSlotMapActivated(ctx, version)
}

func (s *Service) GetSlotMap(ctx context.Context, version *int32, includeSlots bool) (shardadmin.SlotMapVersionDTO, error) {
	return shardadmin.GetSlotMap(ctx, s.GetPool(), version, includeSlots)
}

func (s *Service) CreateSlotMapVersion(ctx context.Context, adminID uuid.UUID, baseVersion *int32, overrides []domain.SlotOverride) (int32, error) {
	return shardadmin.CreateSlotMapVersion(ctx, s, adminID, baseVersion, overrides)
}

func (s *Service) MarkSlotMapMigrating(ctx context.Context, adminID uuid.UUID, version int32, slots []int16, targetShard int16) error {
	return shardadmin.MarkSlotMapMigrating(ctx, s, adminID, version, slots, targetShard)
}

func (s *Service) ActivateSlotMapVersion(ctx context.Context, adminID uuid.UUID, version int32) error {
	return shardadmin.ActivateSlotMapVersionWithMigration(ctx, s, adminID, version)
}

func (s *Service) ActivateSlotMapVersionWithMigration(ctx context.Context, adminID uuid.UUID, version int32) error {
	return shardadmin.ActivateSlotMapVersionWithMigration(ctx, s, adminID, version)
}

func (s *Service) GetSlotMigrations(ctx context.Context, version int32) ([]shardadmin.SlotMigrationDTO, error) {
	return shardadmin.GetSlotMigrations(ctx, s.GetPool(), version)
}

func (s *Service) EnsureSlotMigrationJobs(ctx context.Context, draftVersion int32) error {
	return shardadmin.EnsureSlotMigrationJobs(ctx, s, draftVersion)
}

func (s *Service) CopySlotMigrationData(ctx context.Context, version int32, slot int16) error {
	return shardadmin.CopySlotMigrationData(ctx, s, version, slot)
}

func (s *Service) CopyAllMigratingSlots(ctx context.Context, draftVersion int32) error {
	return shardadmin.CopyAllMigratingSlots(ctx, s, draftVersion)
}

func (s *Service) DrainMigratingSlots(ctx context.Context, version int32) error {
	return shardadmin.DrainMigratingSlots(ctx, s, version)
}

func (s *Service) RollbackSlotMapVersion(ctx context.Context, adminID uuid.UUID, previousVersion int32) error {
	return shardadmin.RollbackSlotMapVersion(ctx, s, adminID, previousVersion)
}

func (s *Service) CatchUpDualWriteSlots(ctx context.Context, draftVersion int32) error {
	return shardadmin.CatchUpDualWriteSlots(ctx, s, draftVersion)
}

func (s *Service) VerifySlotMigrationR5(ctx context.Context) error {
	return shardadmin.VerifySlotMigrationR5(ctx, s)
}

func (s *Service) HasPendingSlotDrain(ctx context.Context) (bool, error) {
	return shardadmin.HasPendingSlotDrain(ctx, s.GetPool())
}

func (s *Service) BumpFencesForPendingMigrations(ctx context.Context) error {
	return shardadmin.BumpFencesForPendingMigrations(ctx, s)
}

func (s *Service) PublishRtbCatalogReload(ctx context.Context) error {
	return shardadmin.PublishControlChannelToAllShards(ctx, s.redisShards, domain.RtbCatalogReloadChannel(s.cfg), "reload")
}

func (s *Service) DedupAdapter(ctx context.Context) *dedup.Adapter {
	if s == nil || s.pool == nil || s.cfg == nil {
		return nil
	}
	return shardadmin.DedupAdapter(ctx, s.pool, s.cfg.RegionCode)
}

func (s *Service) OperationLeaseWorker() *shardadmin.OperationLeaseWorker {
	if s == nil {
		return nil
	}
	return s.leaseWorker
}

func NewOperationLeaseWorker(svc *Service) *shardadmin.OperationLeaseWorker {
	return shardadmin.NewOperationLeaseWorker(svc)
}

func (s *Service) ControlRedis() redis.UniversalClient {
	return shardadmin.PickHealthyControlShard(s.redisShards)
}

func (s *Service) LeaseWorkerConfig() shardadmin.LeaseWorkerConfig {
	cfg := shardadmin.LeaseWorkerConfig{NodeRole: "management"}
	if s == nil || s.cfg == nil {
		return cfg
	}
	cfg.NodeID = s.cfg.NodeID
	if s.cfg.NodeRole != "" {
		cfg.NodeRole = s.cfg.NodeRole
	}
	cfg.RegionCode = int16(s.cfg.RegionCode)
	cfg.OpLeaseTimeoutSec = s.cfg.OpLeaseTimeoutSec
	cfg.OpLeaseMaxRenewals = int32(s.cfg.OpLeaseMaxRenewals)
	cfg.OpLeaseFencingDir = s.cfg.OpLeaseFencingDir
	return cfg
}

func (s *Service) LeaseReplicaNodes() []string {
	if s != nil && s.cfg != nil && s.cfg.NodeID != "" {
		return []string{s.cfg.NodeID}
	}
	return []string{"management"}
}

func (s *Service) GetShardHealth(ctx context.Context) (shardadmin.ShardHealthReport, error) {
	return shardadmin.GetShardHealth(ctx, s)
}

func (s *Service) PublishRoutingCutover(ctx context.Context, routingEpoch int64, slotVersion int32) {
	s.publishRoutingCutover(ctx, routingEpoch, slotVersion)
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
