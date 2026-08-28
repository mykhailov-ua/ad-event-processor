package outbox

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/supply"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Alerter interface {
	OutboxStuckThresholdSec() int
	AlertOutboxStuck(ctx context.Context, pending int64, oldestSeconds float64)
}

type Host interface {
	Pool() *pgxpool.Pool
	WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error
	Config() *config.Config
	OutboxAlerter() Alerter
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	AuditActorID(ctx context.Context) uuid.UUID
	RedisShards() []redis.UniversalClient
	RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	CampaignUpdateChannel() string
	PublishRegistryFullSync(ctx context.Context) error
	PublishRtbCatalogReload(ctx context.Context) error
	FlowReloadChannel() string
	SyncUserConsentToRedis(ctx context.Context, userIDHash string, purposes int16) error
	PurgeUserDataRedis(ctx context.Context, userIDHash, subjectUserID string) error
	MarkErasureRedisPurgeDone(ctx context.Context, erasureID uuid.UUID, purgeErr error) error
	ExportSupplyFiles(ctx context.Context) error
	ApplyQuotaRepair(ctx context.Context, eventID int64, payload []byte) error
	ApplyReconciliationAdjust(ctx context.Context, eventID int64, payload []byte) error
	HandleTelegramEvent(ctx context.Context, payload []byte) error
}

type (
	QuotaRepairPayload           = governance.QuotaRepairPayload
	ReconciliationAdjustPayload  = reconciliation.ReconciliationAdjustPayload
	SupplyFilesPayload           = supply.FilesPayload
)
