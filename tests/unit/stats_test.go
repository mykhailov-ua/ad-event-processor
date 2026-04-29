package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/repository"
	"github.com/stretchr/testify/assert"
)

type MockStatsRepo struct {
	repository.Querier
	mu      sync.Mutex
	updates []repository.UpdateCampaignStatsBatchParams
}

func (m *MockStatsRepo) UpdateCampaignStatsBatch(ctx context.Context, arg repository.UpdateCampaignStatsBatchParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, arg)
	return nil
}

func TestAggregator_Flow(t *testing.T) {
	mock := &MockStatsRepo{}
	// Short flush interval for testing
	agg := ads.NewAggregator(mock, 50*time.Millisecond, 1*time.Second, 1)

	ctx, cancel := context.WithCancel(context.Background())
	agg.Start(ctx)

	id1 := uuid.New()
	agg.Increment(id1, "impression")
	agg.Increment(id1, "impression")
	agg.Increment(id1, "click")

	id2 := uuid.New()
	agg.Increment(id2, "conversion")

	// Wait for flush
	assert.Eventually(t, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return len(mock.updates) > 0
	}, 500*time.Millisecond, 10*time.Millisecond)

	cancel()
	agg.Stop()

	// Verify totals across all updates
	var totalImps, totalClicks, totalConvs int64
	for _, u := range mock.updates {
		for i := range u.CampaignIds {
			totalImps += u.Impressions[i]
			totalClicks += u.Clicks[i]
			totalConvs += u.Conversions[i]
		}
	}

	assert.Equal(t, int64(2), totalImps)
	assert.Equal(t, int64(1), totalClicks)
	assert.Equal(t, int64(1), totalConvs)
}
