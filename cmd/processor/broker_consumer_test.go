package main

import (
	"context"
	"sync"
	"testing"
	"time"

	bserver "ad-event-processor/internal/broker"
	"ad-event-processor/internal/domain"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/ingest/pb"

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

func TestBrokerConsumerGroup_OnMessageProcessed(t *testing.T) {
	tmpDir := t.TempDir()

	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ch-callback-test"

	bp, err := ingestion.NewBrokerProducer(ingestion.BrokerProducerConfig{
		Topic:      topic,
		BrokerAddr: addr,
		BatchSize:  10,
	})
	require.NoError(t, err)

	campaignID := uuid.New()
	for range 3 {
		evt := &domain.Event{
			ClickID:    uuid.New().String(),
			CampaignID: campaignID,
			Type:       "click",
			CreatedAt:  time.Now(),
		}
		require.NoError(t, bp.Enqueue(evt))
	}
	require.NoError(t, bp.Close())

	var seen int
	mockStore := &mockEventStore{}
	bcg, err := NewBrokerConsumerGroup(mockStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "ch_callback_test",
		PartitionCount: 1,
		BatchSize:      10,
		FlushInterval:  20 * time.Millisecond,
		DataDir:        tmpDir,
		OnMessageProcessed: func(evt *domain.Event, _ uint64) {
			if evt != nil && evt.CampaignID == campaignID {
				seen++
			}
		},
	}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	bcg.Start(ctx)

	require.Eventually(t, func() bool {
		return seen >= 3
	}, 3*time.Second, 50*time.Millisecond)

	cancel()
	require.NoError(t, bcg.Wait(context.Background()))
}

func TestBrokerConsumerGroup_BatchFetchAndOffsetCommit(t *testing.T) {
	tmpDir := t.TempDir()

	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ch-events-test"

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

	committed, err := bcg.OffsetTracker().GetCommittedOffset(topic, 0, "ch_consumer_test")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, committed, uint64(5))
}

func TestBrokerConsumerGroup_SpoolAndCatchupOnOutage(t *testing.T) {
	tmpDir := t.TempDir()

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

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, failingStore.Count())

	cancel()
	_ = bcg.Wait(context.Background())

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
		t.Skip("integration: 50k broker batch (run make test-integration)")
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

func TestBrokerConsumerGroup_FraudTopic(t *testing.T) {
	tmpDir := t.TempDir()

	srv := bserver.NewServer("127.0.0.1:0", t.TempDir(), 1024*1024, 4096)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	addr := srv.Addr()
	topic := "ad-fraud-events-test"

	sink, err := ingestion.NewFraudBrokerSink(addr, "", topic, 5*time.Second)
	require.NoError(t, err)
	defer sink.Close()

	evt := &domain.Event{
		ClickID:     "fraud-click",
		CampaignID:  uuid.New(),
		Type:        "click",
		FraudReason: "geo_block",
		FraudScore:  90,
	}
	payloads := make([][]byte, 0, 1)
	payloads = append(payloads, mustMarshalFraudStreamEvent(t, evt))
	require.NoError(t, sink.Produce(context.Background(), 0, payloads))

	mockStore := &mockEventStore{}
	bcg, err := NewBrokerConsumerGroup(mockStore, BrokerConsumerGroupConfig{
		BrokerAddr:     addr,
		Topic:          topic,
		Group:          "fraud_broker_test",
		PartitionCount: 1,
		BatchSize:      10,
		FlushInterval:  50 * time.Millisecond,
		MaxBytes:       512 * 1024,
		DataDir:        tmpDir,
	}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	bcg.Start(ctx)

	require.Eventually(t, func() bool {
		return mockStore.Count() >= 1
	}, 10*time.Second, 50*time.Millisecond)

	cancel()
	require.NoError(t, bcg.Wait(context.Background()))
}

func mustMarshalFraudStreamEvent(t *testing.T, evt *domain.Event) []byte {
	t.Helper()
	pbEvt := &pb.AdStreamEvent{
		ClickId:     []byte(evt.ClickID),
		CampaignId:  evt.CampaignID[:],
		EventType:   []byte(evt.Type),
		FraudReason: []byte(evt.FraudReason),
		FraudScore:  evt.FraudScore,
	}
	data, err := pbEvt.MarshalVT()
	require.NoError(t, err)
	return data
}
