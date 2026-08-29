package testutil

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	ingestion "ad-event-processor/internal/ingest"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type FilterChecker interface {
	PreloadScripts(ctx context.Context) error
	Check(ctx context.Context, evt *domain.Event) error
}

type mgmtTestRegistry struct{}

func (r mgmtTestRegistry) Exists(uuid.UUID) bool { return true }
func (r mgmtTestRegistry) Add(uuid.UUID, uuid.UUID, *uuid.UUID, string, domain.PacingMode, int64, string, int32, int32, []string) {
}
func (r mgmtTestRegistry) GetCustomerID(uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, true }
func (r mgmtTestRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	cp := &domain.Campaign{ID: id, CustomerID: uuid.New(), Location: time.UTC}
	cp.IDStr = id.String()
	cp.BudgetCampaignKey = domain.BudgetCampaignKey(id)
	cp.CampaignSyncKey = domain.CampaignSyncKey(id)
	cp.CustomerSyncKey = "budget:sync:customer:" + cp.CustomerID.String()
	return cp, true
}
func (r mgmtTestRegistry) Sync(context.Context) (int, error)        { return 0, nil }
func (r mgmtTestRegistry) StartSync(context.Context, time.Duration) {}
func (r mgmtTestRegistry) Wait(context.Context) error               { return nil }

func MgmtTestRegistry() domain.CampaignRegistry {
	return mgmtTestRegistry{}
}

func NewLuaUnifiedFilter(redisClient redis.UniversalClient, registry domain.CampaignRegistry) FilterChecker {
	if registry == nil {
		registry = MgmtTestRegistry()
	}
	return ingestion.NewUnifiedFilter(
		[]redis.UniversalClient{redisClient},
		domain.NewJumpHashSharder(1),
		registry,
		nil,
		10_000, time.Minute, time.Hour, time.Hour,
		100_000, 10_000, "events", 10_000,
	)
}
