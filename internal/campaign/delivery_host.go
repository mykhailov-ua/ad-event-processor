package campaign

import (
	"context"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryHost interface {
	RunWithPostgresLow(ctx context.Context, fn func(context.Context) error) error
	Pool() *pgxpool.Pool
	PacingToleranceMargin() float64
	CampaignLocation(timezone string) *time.Location
	PacingHourWeights(ctx context.Context) [24]float64
	AuditPacingLoopAdjustment(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldPacing, newPacing, spend, expected string)

	AutoscaleEnabled() bool
	AutoscaleHighCTRThreshold() float64
	AutoscaleLowCTRThreshold() float64
	AutoscaleMinImpressions() int64
	AutoscaleMinRemainingBudget() int64
	AutoscaleShiftAmount() int64
	AuditAutoscaleBudgetTransfer(ctx context.Context, q db.Querier, campaignID uuid.UUID, change AutoscaleBudgetAuditChange)

	MABMinImpressions() int64
	MABLookbackDays() int
	QueryMABCreativeStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]CreativeMABStat, error)
	QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
		map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		error,
	)

	EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

type WorkerEffects = DeliveryHost

type CreativeMABStat struct {
	Impressions int64
	Clicks      int64
}

type AutoscaleBudgetAuditChange struct {
	OldBudget string
	NewBudget string
	CTR       float64
	Target    string
	Source    string
}

type campaignBudgetOutboxPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}
