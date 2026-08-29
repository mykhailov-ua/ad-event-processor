package budget

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type BudgetDeltaAggregator struct {
	mu      sync.Mutex
	pending map[uuid.UUID]int64
	flushed map[uuid.UUID]int64
}

func NewBudgetDeltaAggregator() *BudgetDeltaAggregator {
	return &BudgetDeltaAggregator{
		pending: make(map[uuid.UUID]int64, 256),
		flushed: make(map[uuid.UUID]int64, 256),
	}
}

func (a *BudgetDeltaAggregator) Record(campaignID uuid.UUID, amountMicro int64) {
	if a == nil || amountMicro == 0 {
		return
	}
	a.mu.Lock()
	a.pending[campaignID] += amountMicro
	if a.pending[campaignID] == 0 {
		delete(a.pending, campaignID)
	}
	a.mu.Unlock()
}

func (a *BudgetDeltaAggregator) PendingDeltaMicro(_ context.Context, campaignID uuid.UUID) (int64, error) {
	if a == nil {
		return 0, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pending[campaignID], nil
}

func (a *BudgetDeltaAggregator) MarkFlushed(campaignID uuid.UUID, amountMicro int64) {
	if a == nil || amountMicro <= 0 {
		return
	}
	a.mu.Lock()
	a.pending[campaignID] -= amountMicro
	if a.pending[campaignID] <= 0 {
		delete(a.pending, campaignID)
	}
	a.flushed[campaignID] += amountMicro
	a.mu.Unlock()
}
