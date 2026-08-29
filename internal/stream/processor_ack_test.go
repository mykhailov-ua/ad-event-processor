package stream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type MockEventStore struct {
	mu      sync.Mutex
	flushes [][]*domain.Event
	Err     error
}

func (m *MockEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	batchCopy := make([]*domain.Event, len(events))
	copy(batchCopy, events)
	m.flushes = append(m.flushes, batchCopy)
	return nil
}

func (m *MockEventStore) Close() error { return nil }

type mockRedisClientAck struct {
	redis.UniversalClient
	xAckFunc func(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd
}

func (m *mockRedisClientAck) XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd {
	if m.xAckFunc != nil {
		return m.xAckFunc(ctx, stream, group, ids...)
	}
	return redis.NewIntCmd(ctx)
}

func TestStreamConsumer_FlushBatch_XAckError(t *testing.T) {
	mockStore := &MockEventStore{}
	mockRdb := &mockRedisClientAck{
		xAckFunc: func(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx)
			cmd.SetErr(errors.New("mock XAck error"))
			return cmd
		},
	}

	p := &StreamConsumer{
		store:        mockStore,
		redisClient:  mockRdb,
		streamName:   "test-stream",
		groupName:    "test-group",
		writeTimeout: 10 * time.Second,
	}

	batch := []*domain.Event{{CampaignID: uuid.New(), Type: "click"}}
	msgIDs := []string{"1-0"}

	err := p.flushBatch(context.Background(), batch, msgIDs, "test-worker")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock XAck error")

	mockStore.mu.Lock()
	flushesCount := len(mockStore.flushes)
	mockStore.mu.Unlock()
	assert.Equal(t, 1, flushesCount)
}
