package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	bserver "github.com/bidshard/ad-event-processor/pkg/broker/server"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEventStore struct {
	mu           sync.Mutex
	storedEvents []*domain.Event
	failWrites   bool
}

func (m *mockEventStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrites {
		return assert.AnError
	}
	m.storedEvents = append(m.storedEvents, events...)
	return nil
}

func (m *mockEventStore) Close() error {
	return nil
}

func (m *mockEventStore) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.storedEvents)
}

func TestBrokerConsumerGroup_BatchFetchAndOffsetCommit(t *testing.T) {
	tmpDir := t.TempDir()

	// Start a local broker server
	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ch-events-test"

	// Produce 500 events using BrokerProducer
	bp, err := ingestion.NewBrokerProducer(ingestion.BrokerProducerConfig{
		Topic:      topic,
		BrokerAddr: addr,
		BatchSize:  100,
	})
	require.NoError(t, err)

	for range 500 {
		evt := &domain.Event{
			ClickID:    uuid.New().String(),
			CampaignID: uuid.New(),
			Type:       "click",
			CreatedAt:  time.Now(),
		}
		require.NoError(t, bp.Enqueue(evt))
	}
	require.NoError(t, bp.Close())

	// Start BrokerConsumerGroup
	mockStore := &mockEventStore{}
	bcg, err := NewBrokerConsumerGroup(mockStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "ch_consumer_test",
		PartitionCount: 1,
		BatchSize:      100,
		FlushInterval:  20 * time.Millisecond,
		DataDir:        tmpDir,
	}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	bcg.Start(ctx)

	require.Eventually(t, func() bool {
		return mockStore.Count() >= 500
	}, 3*time.Second, 50*time.Millisecond)

	cancel()
	require.NoError(t, bcg.Wait(context.Background()))

	assert.GreaterOrEqual(t, mockStore.Count(), 500)

	// Verify offset persistence on disk (5 batch produce messages committed = offset 5)
	committed, err := bcg.OffsetTracker().GetCommittedOffset(topic, 0, "ch_consumer_test")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, committed, uint64(5))
}

func TestBrokerConsumerGroup_SpoolAndCatchupOnOutage(t *testing.T) {
	tmpDir := t.TempDir()

	// Start local broker server
	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ch-outage-test"

	bp, err := ingestion.NewBrokerProducer(ingestion.BrokerProducerConfig{
		Topic:      topic,
		BrokerAddr: addr,
		BatchSize:  50,
	})
	require.NoError(t, err)

	for range 200 {
		evt := &domain.Event{
			ClickID:    uuid.New().String(),
			CampaignID: uuid.New(),
			Type:       "click",
			CreatedAt:  time.Now(),
		}
		require.NoError(t, bp.Enqueue(evt))
	}
	require.NoError(t, bp.Close())

	// Store configured to fail (simulating CH outage)
	failingStore := &mockEventStore{failWrites: true}
	bcg, err := NewBrokerConsumerGroup(failingStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "ch_outage_group",
		PartitionCount: 1,
		BatchSize:      50,
		FlushInterval:  20 * time.Millisecond,
		DataDir:        tmpDir,
	}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	bcg.Start(ctx)

	// Wait 200ms during outage; store should remain empty because writes fail
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, failingStore.Count())

	cancel()
	_ = bcg.Wait(context.Background())

	// Now simulate ClickHouse recovery: store allows writes
	failingStore.mu.Lock()
	failingStore.failWrites = false
	failingStore.mu.Unlock()

	bcg2, err := NewBrokerConsumerGroup(failingStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "ch_outage_group",
		PartitionCount: 1,
		BatchSize:      50,
		FlushInterval:  20 * time.Millisecond,
		DataDir:        tmpDir,
	}, nil)
	require.NoError(t, err)

	ctx2, cancel2 := context.WithCancel(context.Background())
	bcg2.Start(ctx2)

	require.Eventually(t, func() bool {
		return failingStore.Count() >= 200
	}, 3*time.Second, 50*time.Millisecond)

	cancel2()
	require.NoError(t, bcg2.Wait(context.Background()))

	assert.GreaterOrEqual(t, failingStore.Count(), 200)
}

func TestBrokerConsumerGroup_Batch50k(t *testing.T) {
	if testing.Short() {
		t.Skip("50k broker batch integration test")
	}

	tmpDir := t.TempDir()
	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 64*1024*1024, 65536)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ch-batch-50k"
	const total = 50_000

	bp, err := ingestion.NewBrokerProducer(ingestion.BrokerProducerConfig{
		Topic:         topic,
		BrokerAddr:    addr,
		Capacity:      65536,
		BatchSize:     100,
		FlushInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	for range total {
		evt := &domain.Event{
			ClickID:    uuid.New().String(),
			CampaignID: uuid.New(),
			Type:       "click",
			CreatedAt:  time.Now(),
		}
		require.NoError(t, bp.Enqueue(evt))
	}
	require.NoError(t, bp.Close())

	mockStore := &mockEventStore{}
	bcg, err := NewBrokerConsumerGroup(mockStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "ch_batch_50k",
		PartitionCount: 1,
		BatchSize:      10_000,
		FlushInterval:  50 * time.Millisecond,
		MaxBytes:       512 * 1024,
		DataDir:        tmpDir,
	}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	bcg.Start(ctx)

	require.Eventually(t, func() bool {
		return mockStore.Count() >= total
	}, 60*time.Second, 100*time.Millisecond)

	cancel()
	require.NoError(t, bcg.Wait(context.Background()))
	assert.GreaterOrEqual(t, mockStore.Count(), total)
}
