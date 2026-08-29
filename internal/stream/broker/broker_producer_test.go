package broker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBrokerClient struct {
	producedCount atomic.Uint64
	bytesCount    atomic.Uint64
	produceErr    error
}

func (m *mockBrokerClient) Produce(ctx context.Context, topic string, partition uint16, payload []byte) (uint64, error) {
	if m.produceErr != nil {
		return 0, m.produceErr
	}
	m.bytesCount.Add(uint64(len(payload)))
	m.producedCount.Add(1)
	return m.producedCount.Load(), nil
}

func monotonicNano() int64 { return time.Now().UnixNano() }

func percentileDuration(samples []time.Duration, pct int) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), samples...)
	for i := range len(sorted) {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	idx := (len(sorted) - 1) * pct / 100
	return sorted[idx]
}

func (m *mockBrokerClient) Close() error {
	return nil
}

func TestBrokerProducer_ZeroAlloc(t *testing.T) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-zero-alloc",
		Capacity:      1024,
		BatchSize:     64,
		FlushInterval: 100 * time.Millisecond,
		Client:        mockCli,
	})
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	evt := domain.Event{
		ClickID:     "018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f",
		CampaignID:  uuid.MustParse("018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f"),
		Type:        "click",
		UserID:      "user-12345",
		IP:          "192.168.1.100",
		UA:          "Mozilla/5.0 (X11; Linux x86_64)",
		FraudReason: "none",
		FraudScore:  0,
		CreatedAt:   time.Unix(1700000000, 0),
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = bp.Enqueue(&evt)
	})

	assert.Equal(t, float64(0), allocs, "Enqueue on hot path must produce 0 heap allocations")
}

func TestBrokerProducer_LatencySLA(t *testing.T) {
	const (
		warmup     = 2000
		iterations = 10000
		p99Budget  = time.Microsecond
	)

	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-latency-sla",
		Capacity:      65536,
		BatchSize:     512,
		FlushInterval: 5 * time.Millisecond,
		Client:        mockCli,
	})
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	evt := domain.Event{
		ClickID:     "018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f",
		CampaignID:  uuid.MustParse("018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f"),
		Type:        "click",
		UserID:      "user-12345",
		IP:          "192.168.1.100",
		UA:          "Mozilla/5.0 (X11; Linux x86_64)",
		FraudReason: "none",
		FraudScore:  0,
		CreatedAt:   time.Unix(1700000000, 0),
	}

	for range warmup {
		_ = bp.Enqueue(&evt)
	}

	latencies := make([]time.Duration, 0, iterations)
	for range iterations {
		start := monotonicNano()
		require.NoError(t, bp.Enqueue(&evt))
		latencies = append(latencies, time.Duration(monotonicNano()-start))
	}

	p50 := percentileDuration(latencies, 50)
	p99 := percentileDuration(latencies, 99)
	t.Logf("BrokerProducer.Enqueue n=%d p50=%v p99=%v", iterations, p50, p99)
	require.Less(t, p99, p99Budget, "broker Enqueue p99 must stay under 1 us hot-path budget")
}

func TestBrokerProducer_EnqueueAndFlush(t *testing.T) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-flush",
		Capacity:      256,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
		Client:        mockCli,
	})
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	n := 25
	for i := range n {
		evt := &domain.Event{
			ClickID:    fmt.Sprintf("click-%d", i),
			CampaignID: uuid.New(),
			Type:       "impression",
			CreatedAt:  time.Now(),
		}
		require.NoError(t, bp.Enqueue(evt))
	}

	require.Eventually(t, func() bool {
		return mockCli.producedCount.Load() > 0
	}, 1*time.Second, 5*time.Millisecond)

	assert.Greater(t, mockCli.bytesCount.Load(), uint64(0))
}

func TestBrokerProducer_StreamEventEnqueue(t *testing.T) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-stream-evt",
		Capacity:      256,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
		Client:        mockCli,
	})
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	streamEvt := &pb.AdStreamEvent{
		ClickId:       []byte("click-stream-1"),
		CampaignId:    []byte("1234567890123456"),
		EventType:     []byte("click"),
		CreatedAtUnix: time.Now().Unix(),
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = bp.EnqueueStreamEvent(streamEvt)
	})
	assert.Equal(t, float64(0), allocs, "EnqueueStreamEvent must produce 0 heap allocations")

	require.Eventually(t, func() bool {
		return mockCli.producedCount.Load() > 0
	}, 1*time.Second, 5*time.Millisecond)
}

func TestBrokerProducer_RingOverflow(t *testing.T) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-overflow",
		Capacity:      4,
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
		Client:        mockCli,
	})
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	evt := &domain.Event{
		ClickID:   "c-1",
		CreatedAt: time.Now(),
	}

	filled := 0
	for range 10 {
		if err := bp.Enqueue(evt); err == nil {
			filled++
		}
	}

	assert.Equal(t, 4, filled)
	assert.Greater(t, bp.DroppedCount(), uint64(0))
}

func TestBrokerProducer_ShutdownDrain(t *testing.T) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "test-drain",
		Capacity:      128,
		BatchSize:     32,
		FlushInterval: 1 * time.Hour,
		Client:        mockCli,
	})
	require.NoError(t, err)

	for i := range 50 {
		_ = bp.Enqueue(&domain.Event{
			ClickID:   fmt.Sprintf("click-drain-%d", i),
			CreatedAt: time.Now(),
		})
	}

	require.NoError(t, bp.Close())

	assert.Greater(t, mockCli.producedCount.Load(), uint64(0))
	assert.Equal(t, ErrProducerClosed, bp.Enqueue(&domain.Event{}))
}

func BenchmarkTrackerToBroker(b *testing.B) {
	mockCli := &mockBrokerClient{}
	bp, err := NewBrokerProducer(BrokerProducerConfig{
		Topic:         "bench-tracker-to-broker",
		Capacity:      65536,
		BatchSize:     512,
		FlushInterval: 5 * time.Millisecond,
		Client:        mockCli,
	})
	if err != nil {
		b.Fatalf("failed to create broker producer: %v", err)
	}
	defer func() { _ = bp.Close() }()

	evt := domain.Event{
		ClickID:     "018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f",
		CampaignID:  uuid.MustParse("018f3a5e-7a2b-7c1d-8e9f-0a1b2c3d4e5f"),
		Type:        "click",
		UserID:      "user-bench-123",
		IP:          "10.0.0.1",
		UA:          "Mozilla/5.0",
		FraudReason: "clean",
		FraudScore:  0,
		CreatedAt:   time.Unix(1700000000, 0),
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := bp.Enqueue(&evt); err != nil {
			time.Sleep(10 * time.Microsecond)
		}
	}
}

func TestBrokerProducerSet_notPinnedToPartition0(t *testing.T) {
	const parts = 6
	list := make([]*BrokerProducer, parts)
	for i := range parts {
		bp, err := NewBrokerProducer(BrokerProducerConfig{
			Topic:         "test-fanout",
			Partition:     uint16(i),
			Capacity:      256,
			BatchSize:     32,
			FlushInterval: 2 * time.Millisecond,
			Client:        &mockBrokerClient{},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = bp.Close() })
		list[i] = bp
	}
	set := NewBrokerProducerSet(list)
	require.Equal(t, parts, set.Len())

	hits := make([]int, parts)
	for range 256 {
		id := uuid.New()
		idx, bp := set.Pick(id)
		require.NotNil(t, bp)
		require.GreaterOrEqual(t, idx, 0)
		require.Less(t, idx, parts)
		hits[idx]++
		require.NoError(t, bp.Enqueue(&domain.Event{
			CampaignID: id,
			ClickID:    "c",
			CreatedAt:  time.Now(),
		}))
	}
	used := 0
	for _, n := range hits {
		if n > 0 {
			used++
		}
	}
	if used < 2 {
		t.Fatalf("broker produce pinned to one partition, hits=%v", hits)
	}
	if hits[0] == 256 {
		t.Fatal("all campaigns routed to partition 0")
	}
}
