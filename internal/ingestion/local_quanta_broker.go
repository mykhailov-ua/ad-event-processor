package ingestion

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"
	"github.com/bidshard/ad-event-processor/pkg/broker/client"

	"github.com/google/uuid"
)

const (
	budgetDeltaRingCapacity = 8192
	budgetDeltaRingMask     = budgetDeltaRingCapacity - 1
	budgetDeltaRingUsable   = budgetDeltaRingCapacity - 1
	budgetDeltaFlushBatch   = 256
	budgetDeltaFlushEvery   = 5 * time.Millisecond
)

type budgetDeltaSlot struct {
	ready       atomic.Uint32
	amountMicro int64
	campaignID  uuid.UUID
}

type BudgetDeltaPublisher struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       [budgetDeltaRingCapacity]budgetDeltaSlot

	topic     string
	trackerID []byte
	cli       *client.Client
	timeout   time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type BudgetDeltaPublisherConfig struct {
	BrokerAddr string
	RedisURL   string
	Topic      string
	TrackerID  string
	Timeout    time.Duration
}

func NewBudgetDeltaPublisher(cfg BudgetDeltaPublisherConfig) *BudgetDeltaPublisher {
	if cfg.BrokerAddr == "" || cfg.Topic == "" {
		return nil
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	p := &BudgetDeltaPublisher{
		topic:     cfg.Topic,
		trackerID: []byte(cfg.TrackerID),
		cli:       client.NewClient(cfg.BrokerAddr, cfg.Timeout),
		timeout:   cfg.Timeout,
		stopCh:    make(chan struct{}),
	}
	if cfg.RedisURL != "" {
		p.cli.SetRedisURL(cfg.RedisURL)
	}
	p.wg.Add(1)
	go p.worker()
	return p
}

func (p *BudgetDeltaPublisher) Publish(campaignID uuid.UUID, amountMicro int64) {
	if p == nil || amountMicro <= 0 {
		return
	}
	p.enqueue(campaignID, amountMicro)
}

func (p *BudgetDeltaPublisher) PublishReturn(campaignID uuid.UUID, amountMicro int64) {
	if p == nil || amountMicro <= 0 {
		return
	}
	p.enqueue(campaignID, -amountMicro)
}

func (p *BudgetDeltaPublisher) enqueue(campaignID uuid.UUID, amountMicro int64) {
	for {
		alloc := atomic.LoadUint64(&p.allocCursor)
		read := atomic.LoadUint64(&p.readCursor)
		if alloc-read >= budgetDeltaRingUsable {
			return
		}
		if !atomic.CompareAndSwapUint64(&p.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & budgetDeltaRingMask
		slot := &p.slots[idx]
		if slot.ready.Load() != 0 {
			return
		}
		slot.campaignID = campaignID
		slot.amountMicro = amountMicro
		slot.ready.Store(1)
		atomic.StoreUint64(&p.writeCursor, alloc+1)
		return
	}
}

func (p *BudgetDeltaPublisher) Close() {
	if p == nil {
		return
	}
	close(p.stopCh)
	p.wg.Wait()
	_ = p.cli.Close()
}

func (p *BudgetDeltaPublisher) worker() {
	defer p.wg.Done()
	if err := p.cli.Connect(); err != nil {
		return
	}
	ticker := time.NewTicker(budgetDeltaFlushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			p.flush()
			return
		case <-ticker.C:
			p.flush()
		}
	}
}

func (p *BudgetDeltaPublisher) flush() {
	batch := 0
	for batch < budgetDeltaFlushBatch {
		read := atomic.LoadUint64(&p.readCursor)
		write := atomic.LoadUint64(&p.writeCursor)
		if read >= write {
			return
		}
		idx := read & budgetDeltaRingMask
		slot := &p.slots[idx]
		if slot.ready.Load() != 1 {
			return
		}
		p.produce(slot.campaignID, slot.amountMicro)
		slot.ready.Store(0)
		atomic.StoreUint64(&p.readCursor, read+1)
		batch++
	}
}

func (p *BudgetDeltaPublisher) produce(campaignID uuid.UUID, amountMicro int64) {
	msg := budgetDeltaPool.Get().(*pb.BudgetDelta)
	msg.Reset()
	msg.CampaignId = append(msg.CampaignId[:0], campaignID[:]...)
	msg.AmountMicro = amountMicro
	msg.CreatedAtUnixNano = monotonicNano()
	msg.TrackerId = append(msg.TrackerId[:0], p.trackerID...)
	data, err := msg.MarshalVT()
	budgetDeltaPool.Put(msg)
	if err != nil || len(data) == 0 {
		return
	}
	_, _ = p.cli.Produce(p.topic, 0, data)
}

var budgetDeltaPool = sync.Pool{
	New: func() any { return &pb.BudgetDelta{} },
}

func FetchRecoveryDeltas(ctx context.Context, cfg domain.BrokerConsumerConfig, startOffset uint64) (map[uuid.UUID]int64, error) {
	out := make(map[uuid.UUID]int64)
	if cfg.BrokerAddr == "" || cfg.Topic == "" {
		return out, nil
	}
	cli := client.NewClient(cfg.BrokerAddr, cfg.Timeout)
	if cfg.RedisURL != "" {
		cli.SetRedisURL(cfg.RedisURL)
	}
	if err := cli.Connect(); err != nil {
		return out, err
	}
	defer func() { _ = cli.Close() }()

	offset := startOffset
	for {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		iter, err := cli.Fetch(cfg.Topic, cfg.Partition, offset, cfg.MaxBytes)
		if err != nil {
			return out, err
		}
		n := 0
		for iter.Next() {
			n++
			delta := budgetDeltaPool.Get().(*pb.BudgetDelta)
			delta.Reset()
			if err := delta.UnmarshalVT(iter.Payload); err == nil && len(delta.CampaignId) >= 16 {
				var id uuid.UUID
				_ = ParseUUID(delta.CampaignId[:16], &id)
				out[id] += delta.AmountMicro
			}
			budgetDeltaPool.Put(delta)
			offset = iter.Offset + 1
		}
		if n == 0 {
			return out, nil
		}
	}
}
