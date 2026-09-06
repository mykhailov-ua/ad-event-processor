package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type rollbackProbeFilter struct {
	rollbackCalls int
}

func (f *rollbackProbeFilter) Check(ctx context.Context, evt *domain.Event) error {
	return nil
}

func (f *rollbackProbeFilter) StreamDeferredToProducer() bool { return true }

func (f *rollbackProbeFilter) SetDeferStreamToProducer(deferWrite bool) {}

func (f *rollbackProbeFilter) ClickAmountMicro() int64 { return 100 }

func (f *rollbackProbeFilter) ImpressionAmountMicro() int64 { return 1 }

func (f *rollbackProbeFilter) LocalQuantaFullSkipEligible(evt *domain.Event, camp *domain.Campaign) bool {
	return false
}

func (f *rollbackProbeFilter) RollbackDebit(ctx context.Context, evt *domain.Event, camp *domain.Campaign, debitAmount int64, isLocalQuanta bool) {
	f.rollbackCalls++
}

func (f *rollbackProbeFilter) SetSkipBudgetDebit(skip bool) {}

func TestPublishAcceptedOrRollback_holdout(t *testing.T) {
	probe := &rollbackProbeFilter{}
	engine := NewFilterEngine(0, probe)

	deps := TrackPublishDeps{
		Cfg:          &config.Config{},
		Sharder:      NewJumpHashSharder(1),
		FilterEngine: engine,
		Registry:     &mockRegistry{},
	}

	evt := &domain.Event{
		CampaignID: uuid.New(),
		ClickID:    "holdout-publish-rollback",
		Type:       "click",
	}

	require.False(t, deps.PublishAcceptedOrRollback(context.Background(), evt, nil))
	require.Equal(t, 1, probe.rollbackCalls, "post-debit publish fail must RollbackDebit once")
}

func TestPublishAcceptedOrRollback_holdoutSkipsRollbackOnSuccess(t *testing.T) {
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{"127.0.0.1:1"}})
	defer func() { _ = redisClient.Close() }()

	uf := NewUnifiedFilter([]redis.UniversalClient{redisClient}, nil, nil, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	engine := NewFilterEngine(0, uf)

	p := NewStreamProducer(redisClient, "events", 8, 0)
	defer p.Close()

	deps := TrackPublishDeps{
		Cfg:             &config.Config{},
		Sharder:         NewJumpHashSharder(1),
		StreamProducers: []*StreamProducer{p},
		FilterEngine:    engine,
		Registry:        &mockRegistry{},
	}

	evt := &domain.Event{
		CampaignID: uuid.New(),
		ClickID:    "holdout-publish-success",
		Type:       "click",
	}

	require.True(t, deps.PublishAcceptedOrRollback(context.Background(), evt, nil))
}
