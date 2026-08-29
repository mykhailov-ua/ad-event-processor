package stream

import (
	"context"
	"fmt"
	"sync"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
)

type SpendSyncBatchResult struct {
	Committed uint32
}

type SpendSyncTransport interface {
	ProduceSpendSyncPayload(payload []byte) (SpendSyncBatchResult, error)
}

type SpendSyncProducer struct {
	transport SpendSyncTransport
	minBatch  int

	mu      sync.Mutex
	pending []pendingSpendSync
}

type pendingSpendSync struct {
	worker *domain.SyncWorker
	id     uuid.UUID
	rollup domain.PendingRollup
}

func NewSpendSyncProducer(transport SpendSyncTransport, minBatch int) *SpendSyncProducer {
	if minBatch <= 0 {
		minBatch = 100
	}
	return &SpendSyncProducer{
		transport: transport,
		minBatch:  minBatch,
	}
}

func (p *SpendSyncProducer) PendingCount() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pending)
}

func (p *SpendSyncProducer) EnqueueRollup(ctx context.Context, w *domain.SyncWorker, id uuid.UUID, entry domain.PendingRollup) error {
	if p == nil || p.transport == nil {
		return fmt.Errorf("spend sync producer: unavailable")
	}
	if entry.AmountMicro <= 0 || entry.TxID == "" {
		return nil
	}

	p.mu.Lock()
	p.pending = append(p.pending, pendingSpendSync{worker: w, id: id, rollup: entry})
	shouldFlush := len(p.pending) >= p.minBatch
	p.mu.Unlock()

	if shouldFlush {
		return p.Flush(ctx)
	}
	return nil
}

func (p *SpendSyncProducer) Flush(ctx context.Context) error {
	if p == nil || p.transport == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.Lock()
	if len(p.pending) < p.minBatch {
		p.mu.Unlock()
		return nil
	}
	items := append([]pendingSpendSync(nil), p.pending...)
	p.pending = p.pending[:0]
	p.mu.Unlock()

	txns := make([]dedupkey.SpendSyncTxn, 0, len(items))
	for _, item := range items {
		txns = append(txns, dedupkey.SpendSyncTxn{
			CampaignID:  item.id,
			AmountMicro: item.rollup.AmountMicro,
			TxnID:       item.rollup.TxID,
		})
	}

	payload, err := dedupkey.EncodeSpendSyncPayload(txns)
	if err != nil {
		p.requeueLocked(items)
		return fmt.Errorf("spend sync producer encode batch: %w", err)
	}

	result, err := p.transport.ProduceSpendSyncPayload(payload)
	if err != nil {
		p.requeueLocked(items)
		return fmt.Errorf("spend sync producer produce batch: %w", err)
	}
	if result.Committed == 0 {
		p.requeueLocked(items)
		return fmt.Errorf("spend sync producer produce batch: zero committed")
	}

	metrics.RegionSpendSyncBatchesTotal.Inc()
	metrics.RegionSpendSyncTxnsTotal.Add(float64(len(txns)))
	for _, item := range items {
		if item.worker != nil {
			item.worker.CommitRollupRedis(ctx, item.rollup)
		}
	}
	return nil
}

func (p *SpendSyncProducer) requeueLocked(items []pendingSpendSync) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(items, p.pending...)
}
