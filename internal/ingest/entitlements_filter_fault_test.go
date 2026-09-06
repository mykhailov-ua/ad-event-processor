package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errEntitlementsPipeTracker struct {
	mockPipeliner
}

func (p *errEntitlementsPipeTracker) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

func (p *errEntitlementsPipeTracker) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return nil, redis.ErrClosed
}

type errEntitlementsRedisMock struct {
	mockRedisClient
}

func (m *errEntitlementsRedisMock) Pipeline() redis.Pipeliner {
	return &errEntitlementsPipeTracker{
		mockPipeliner: mockPipeliner{
			incrCmd: redis.NewIntCmd(context.Background()),
			doCmd:   redis.NewCmd(context.Background()),
		},
	}
}

func TestEntitlementsFilter_redisError_failClosed_holdout(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
	}
	reg := benchRegistryForCampaign(camp)
	reg.SeedCustomerEntitlementsForTest(custID, licensing.Entitlements{
		Limits: licensing.Limits{MaxRequestsPerDay: 1000},
	}, licensing.StateActive)

	f := NewEntitlementsFilter(reg, NewJumpHashSharder(1), []redis.UniversalClient{&errEntitlementsRedisMock{}})
	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.80",
	}

	err := f.Check(context.Background(), evt)
	require.Error(t, err)
	kind, ok := classifyFilterErr(err)
	require.True(t, ok)
	assert.Equal(t, filterRejectInfra, kind)
}

func TestEntitlementsFilter_nilShard_failClosed_holdout(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
	}
	reg := benchRegistryForCampaign(camp)
	reg.SeedCustomerEntitlementsForTest(custID, licensing.Entitlements{
		Limits: licensing.Limits{MaxRequestsPerDay: 1000},
	}, licensing.StateActive)

	f := NewEntitlementsFilter(reg, NewJumpHashSharder(1), []redis.UniversalClient{nil})
	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.81",
	}

	err := f.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrShardUnavailable)
}
