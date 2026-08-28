package shardadmin

import (
	"context"

	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type LeaseWorkerConfig struct {
	NodeID             string
	NodeRole           string
	RegionCode         int16
	OpLeaseTimeoutSec  int
	OpLeaseMaxRenewals int32
	OpLeaseFencingDir  string
}

type LeaseHost interface {
	Pool() *pgxpool.Pool
	RedisShards() []redis.UniversalClient
	ControlRedis() redis.UniversalClient
	WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error
	WithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	DedupAdapter(ctx context.Context) *dedup.Adapter
	LeaseWorkerConfig() LeaseWorkerConfig
	LeaseReplicaNodes() []string
}

type CatchupHost interface {
	RedisShards() []redis.UniversalClient
	CampaignUpdateChannel() string
	TryReconnectShard0(ctx context.Context, opts database.RedisShardOptions) bool
}

type HealthHost interface {
	GetSettings(ctx context.Context) (map[string]string, error)
	Pool() *pgxpool.Pool
	RedisShards() []redis.UniversalClient
}

type Host interface {
	Pool() *pgxpool.Pool
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	AlertSlotMapMigrating(ctx context.Context, version int32, slots []int16, targetShard int16)
	AfterSlotMapActivated(ctx context.Context, version int32)
	RedisShards() []redis.UniversalClient
	ListActiveCampaignUUIDs(ctx context.Context) ([]uuid.UUID, error)
	SlotMigrationDualWriteEnabled() bool
	DualWriteConfig() domain.SlotMigrationDualWriteConfig
	MigrationFenceEnabled() bool
	AlertSlotMigrationError(ctx context.Context, stage string, err error)
	CheckStuckDrainJobs(ctx context.Context)
}

type OrchestratorHost interface {
	Pool() *pgxpool.Pool
	RedisShards() []redis.UniversalClient
	ListActiveCampaignUUIDs(ctx context.Context) ([]uuid.UUID, error)
	PublishRoutingCutover(ctx context.Context, routingEpoch int64, slotVersion int32)
	CreateSlotMapVersion(ctx context.Context, adminID uuid.UUID, baseVersion *int32, overrides []domain.SlotOverride) (int32, error)
	MarkSlotMapMigrating(ctx context.Context, adminID uuid.UUID, version int32, slots []int16, targetShard int16) error
	EnsureSlotMigrationJobs(ctx context.Context, draftVersion int32) error
	CopyAllMigratingSlots(ctx context.Context, draftVersion int32) error
	ActivateSlotMapVersion(ctx context.Context, adminID uuid.UUID, version int32) error
	DrainMigratingSlots(ctx context.Context, version int32) error
}
