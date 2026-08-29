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

type countingCampaignRepo struct {
	slowCampaignRepo
	getByIDCalls int
}

func (r *countingCampaignRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	r.getByIDCalls++
	return r.slowCampaignRepo.GetByID(ctx, id)
}

func TestUnifiedFilter_PGFallbackDisabled_NoGetByIDOnCacheMiss(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	reg := &mockRegistry{}
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = campID
		c.CustomerID = custID
		c.IDStr = campID.String()
		c.CustomerIDStr = custID.String()
		c.IDStrAny = campID.String()
		c.CustomerIDStrAny = custID.String()
		c.DailyBudgetMicroAny = int64(10_000_000)
		c.Location = time.UTC
	})

	repo := &countingCampaignRepo{slowCampaignRepo: slowCampaignRepo{delay: 0}}
	f := NewUnifiedFilter(
		[]redis.UniversalClient{&budgetMissRedis{}},
		NewJumpHashSharder(1),
		reg,
		repo,
		1000,
		time.Minute,
		time.Hour,
		time.Hour,
		1_000_000,
		10_000,
		"events-pg-fallback-off",
		10000,
	)
	f.SetPGFallbackAllowed(false)

	beforePG := testutil.ToFloat64(metrics.BudgetCacheMissPostgresTotal)
	err := f.Check(context.Background(), &domain.Event{
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		Type:       "click",
		IP:         "1.1.1.1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
	assert.Equal(t, 0, repo.getByIDCalls)
	assert.Equal(t, beforePG, testutil.ToFloat64(metrics.BudgetCacheMissPostgresTotal))
}
