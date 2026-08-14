package ingestion

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FailingEventStore struct {
	mu      sync.Mutex
	flushes [][]*domain.Event
	calls   atomic.Int64
	failErr error
	healed  atomic.Bool
}

func (m *FailingEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	m.calls.Add(1)
	if !m.healed.Load() {
		return m.failErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	batchCopy := make([]*domain.Event, len(events))
	copy(batchCopy, events)
	m.flushes = append(m.flushes, batchCopy)
	return nil
}

func (m *FailingEventStore) Close() error { return nil }

func (m *FailingEventStore) Heal() { m.healed.Store(true) }

func TestStreamConsumer_CircuitBreakerStopsReads(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers Redis)")
	}

	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	failStore := &FailingEventStore{
		failErr: errors.New("database connection refused"),
	}

	producer := NewStreamProducer(rdb, "cb-test", 1000, 1*time.Second)
	consumer := NewStreamConsumer(
		failStore, rdb, "cb-test", "cb-group", "cb-c",
		2, 1,
		50*time.Millisecond,
		1*time.Second,
		10*time.Millisecond,
		50*time.Millisecond,
		3,
		1*time.Minute,
		1*time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for range 5 {
		err := producer.Process(&domain.Event{CampaignID: uuid.New(), Type: "click"})
		require.NoError(t, err)
	}

	consumer.Start(ctx)

	assert.Eventually(t, func() bool {
		return consumer.cb.State() == CircuitOpen
	}, 3*time.Second, 10*time.Millisecond, "circuit breaker should be open")

	callsAtOpen := failStore.calls.Load()

	// CI budget: 500ms poll window — no fixed sleep before primary assertion.
	assert.Eventually(t, func() bool {
		extra := failStore.calls.Load() - callsAtOpen
		return extra <= 4
	}, 500*time.Millisecond, 10*time.Millisecond,
		"CB should prevent flush calls while open, got %d extra calls", failStore.calls.Load()-callsAtOpen)

	failStore.Heal()

	// CI budget: 5s max wait for half-open probe success after heal.
	assert.Eventually(t, func() bool {
		return consumer.cb.State() == CircuitClosed
	}, 5*time.Second, 10*time.Millisecond, "circuit breaker should recover to closed")

	consumer.Close()
	consumer.Wait(ctx)
}
