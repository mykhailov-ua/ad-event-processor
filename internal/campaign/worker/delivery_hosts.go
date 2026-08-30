package worker

import (
	"context"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryPostgresHost interface {
	RunWithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	Pool() *pgxpool.Pool
}

type PacingDeliveryHost interface {
	PacingToleranceMargin() float64
	CampaignLocation(timezone string) *time.Location
	PacingHourWeights(ctx context.Context) [24]float64
	AuditPacingLoopAdjustment(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldPacing, newPacing, spend, expected string)
}

type AutoscaleDeliveryHost interface {
	AutoscaleEnabled() bool
	AutoscaleHighCTRThreshold() float64
	AutoscaleLowCTRThreshold() float64
	AutoscaleMinImpressions() int64
	AutoscaleMinRemainingBudget() int64
	AutoscaleShiftAmount() int64
	AuditAutoscaleBudgetTransfer(ctx context.Context, q db.Querier, campaignID uuid.UUID, change AutoscaleBudgetAuditChange)
}

type BanditDeliveryHost interface {
	MABMinImpressions() int64
	MABLookbackDays() int
	TrafficOptimizerEnabled() bool
	QueryMABCreativeStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]CreativeMABStat, error)
	QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
		map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		error,
	)
}

type DeliveryPublishHost interface {
	EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

type DeliveryHost interface {
	DeliveryPostgresHost
	PacingDeliveryHost
	AutoscaleDeliveryHost
	BanditDeliveryHost
	DeliveryPublishHost
}

type WorkerEffects = DeliveryHost

type autoscaleDeliveryHost interface {
	DeliveryPostgresHost
	AutoscaleDeliveryHost
}
