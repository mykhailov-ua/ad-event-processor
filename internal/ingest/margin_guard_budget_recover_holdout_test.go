package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pausedStatusCampaignRepo struct {
	camp *domain.Campaign
}

func (r pausedStatusCampaignRepo) GetByID(context.Context, uuid.UUID) (*domain.Campaign, error) {
	return r.camp, nil
}

func (r pausedStatusCampaignRepo) UpdateStatus(context.Context, uuid.UUID, domain.CampaignStatus) error {
	return nil
}

func (r pausedStatusCampaignRepo) UpdateSpend(context.Context, uuid.UUID, int64, string) error {
	return nil
}

func (r pausedStatusCampaignRepo) ListActive(context.Context) ([]*domain.Campaign, error) {
	return nil, nil
}

func TestTryRecoverBudgetFromRegistry_cachedMockCampPausedNoRecover_holdout(t *testing.T) {
	campID := uuid.New()
	camp := &domain.Campaign{
		ID:           campID,
		Status:       domain.CampaignStatusPaused,
		BudgetLimit:  1_000_000,
		CurrentSpend: 200_000,
	}
	enrichMockCampaign(camp)
	cachedMockCamp.Store(camp)
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	redisMock := &budgetRecoverSetRedis{}
	recovered, err := TryRecoverBudgetFromRegistry(context.Background(), redisMock, &mockRegistry{}, campID, camp.BudgetCampaignKey, 0)
	require.NoError(t, err)
	require.False(t, recovered)
}

func TestTryRecoverBudgetFromRegistry_pausedCampaignNoRecover_holdout(t *testing.T) {
	campID := uuid.New()
	reg := NewRegistry(nil)
	camp := &domain.Campaign{
		ID:           campID,
		Status:       domain.CampaignStatusPaused,
		BudgetLimit:  1_000_000,
		CurrentSpend: 200_000,
	}
	enrichMockCampaign(camp)
	reg.SeedCampaignForTest(camp)

	redisMock := &budgetRecoverSetRedis{}
	recovered, err := TryRecoverBudgetFromRegistry(context.Background(), redisMock, reg, campID, camp.BudgetCampaignKey, 0)
	require.NoError(t, err)
	require.False(t, recovered, "paused campaign must not re-warm budget key from registry")
}

func TestUnifiedFilter_budgetMiss_pausedCampaignNoPGRecover_holdout(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:           campID,
		CustomerID:   custID,
		Status:       domain.CampaignStatusPaused,
		BudgetLimit:  10_000_000,
		CurrentSpend: 0,
	}
	enrichMockCampaign(camp)
	cachedMockCamp.Store(camp)
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	repo := pausedStatusCampaignRepo{camp: &domain.Campaign{
		ID:           campID,
		CustomerID:   custID,
		Status:       domain.CampaignStatusPaused,
		BudgetLimit:  10_000_000,
		CurrentSpend: 0,
	}}

	f := NewUnifiedFilter(
		[]redis.UniversalClient{&budgetMissOnceRedis{}},
		NewJumpHashSharder(1),
		&mockRegistry{},
		repo,
		1000,
		time.Minute,
		time.Hour,
		time.Hour,
		1_000_000,
		10_000,
		"events-paused-no-pg-recover",
		10000,
	)
	f.SetPGFallbackAllowed(true)

	beforePG := testutil.ToFloat64(metrics.BudgetCacheMissPostgresTotal)
	beforeRecover := testutil.ToFloat64(metrics.BudgetCacheRegistryRecoverTotal)

	err := f.Check(context.Background(), &domain.Event{
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		Type:       "click",
		IP:         "1.1.1.1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
	assert.Equal(t, beforeRecover, testutil.ToFloat64(metrics.BudgetCacheRegistryRecoverTotal))
	assert.Equal(t, beforePG+1, testutil.ToFloat64(metrics.BudgetCacheMissPostgresTotal))
}
