package ingestion

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *MockEventStore) Close() error {
	return nil
}

func TestStreamConsumer_Ingestion(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	producer := NewStreamProducer(rdb, "s1", 1000, 1*time.Second)

	err := producer.Process(&domain.Event{CampaignID: uuid.New(), Type: "click"})
	assert.NoError(t, err)
}

func TestStreamConsumer_BatchFlushing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers Redis)")
	}
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	mockStore := &MockEventStore{}
	producer := NewStreamProducer(rdb, "s2", 1000, 1*time.Second)
	proc := NewStreamConsumer(mockStore, rdb, "s2", "g2", "c2", 2, 1, 10*time.Second, 1*time.Second, 10*time.Millisecond, 100*time.Millisecond, 3, 1*time.Minute, 1*time.Second)

	proc.Start(context.Background())

	for range 3 {
		_ = producer.Process(&domain.Event{CampaignID: uuid.New(), Type: "click"})
	}

	// CI budget: 5s max wait for batch flush (replaces fixed sleeps).
	require.Eventually(t, func() bool {
		mockStore.mu.Lock()
		defer mockStore.mu.Unlock()
		return len(mockStore.flushes) >= 1
	}, 5*time.Second, 10*time.Millisecond, "StoreBatch flush hook must record at least one batch")

	proc.Close()
	proc.Wait(context.Background())
}

func TestStreamConsumer_DLQ(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	failStore := &FailingEventStore{
		failErr: errors.New("simulated poison pill"),
	}

	producer := NewStreamProducer(rdb, "s_dlq", 1000, 1*time.Second)

	proc := NewStreamConsumer(failStore, rdb, "s_dlq", "g_dlq", "c_dlq", 2, 1, 10*time.Millisecond, 1*time.Second, 10*time.Millisecond, 10*time.Millisecond, 1, 1*time.Minute, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc.Start(ctx)

	for range 2 {
		_ = producer.Process(&domain.Event{CampaignID: uuid.New(), Type: "click"})
	}

	assert.Eventually(t, func() bool {
		size, err := rdb.XLen(ctx, "ad:events:dlq").Result()
		return err == nil && size == 2
	}, 3*time.Second, 50*time.Millisecond, "Should have 2 events in DLQ")

	pending, err := rdb.XPending(ctx, "s_dlq", "g_dlq").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)

	proc.Close()
	proc.Wait(ctx)
}
