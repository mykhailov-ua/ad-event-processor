package reconciliation

import (
	"context"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type BrokerPendingDeltaReader interface {
	// PendingDeltaMicro: spend accepted to broker ring but not yet reflected in Redis budget keys.
	PendingDeltaMicro(ctx context.Context, campaignID uuid.UUID) (int64, error)
}

type PaymentQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Alerter interface {
	AlertReconDiscrepancy(ctx context.Context, runID int64, discrepancies int, totalDelta int64, period string)
	AlertReconDiscrepancyUnresolved(ctx context.Context, runID int64, unresolved int, totalDelta int64, period string, oldest time.Time)
}

type ReconInfraHost interface {
	Pool() *pgxpool.Pool
	SettlementPool() *pgxpool.Pool
	PaymentQueryPool() PaymentQueryer
	RedisShards() []redis.UniversalClient
	Sharder() domain.Sharder
	Config() *config.Config
	// WithPostgresLow: recon worker defers ticks when PG write gate rejects cold-path work.
	WithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	ClickHouseQuery() *database.ClickHouseQuery
	RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient
}

type ReconOpsHost interface {
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	Alerter() Alerter
	BrokerDeltas() BrokerPendingDeltaReader
	InvalidServiceFilterErr() error
	RunStuckDrainCheck(ctx context.Context)
}

type ReconRepairHost interface {
	ForceRefillCampaignFromPG(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error
}

// Host is the controlplane bridge port: infra reads, outbox adjust side effects, optional force-refill repair.
type Host interface {
	ReconInfraHost
	ReconOpsHost
	ReconRepairHost
}

type listRunsHost interface {
	ReconInfraHost
	ReconOpsHost
}
