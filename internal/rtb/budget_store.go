package rtb

import (
	"sync"
	"sync/atomic"
)

const defaultBudgetCap = 10000

type AlignedBudget struct {
	Value int64
	_     [7]int64
}

type budgetSlice struct {
	data []AlignedBudget
}

type BudgetStore struct {
	mu              sync.Mutex
	slots           map[CampaignID]uint32
	customerSlots   map[CustomerID]uint32
	budgets         atomic.Pointer[budgetSlice]
	customerBudgets atomic.Pointer[budgetSlice]
	dailySpent      atomic.Pointer[budgetSlice]
	dailyEpoch      atomic.Uint32
}

func NewBudgetStore() *BudgetStore {
	store := &BudgetStore{
		slots:         make(map[CampaignID]uint32),
		customerSlots: make(map[CustomerID]uint32),
	}
	empty := &budgetSlice{data: make([]AlignedBudget, 0, defaultBudgetCap)}
	store.budgets.Store(empty)
	store.customerBudgets.Store(&budgetSlice{data: make([]AlignedBudget, 0, defaultBudgetCap)})
	store.dailySpent.Store(&budgetSlice{data: make([]AlignedBudget, 0, defaultBudgetCap)})
	return store
}

func (st *BudgetStore) GetOrAllocateSlot(id CampaignID, initialBudget int64) uint32 {
	st.mu.Lock()
	if idx, exists := st.slots[id]; exists {
		st.mu.Unlock()
		return idx
	}
	idx := st.appendSlotLocked(normalizeBudget(initialBudget))
	st.slots[id] = idx
	st.mu.Unlock()
	return idx
}

func (st *BudgetStore) GetOrAllocateCustomerSlot(id CustomerID, initialBudget int64) uint32 {
	if id == 0 {
		return invalidCustomerBudgetIdx
	}
	st.mu.Lock()
	if idx, exists := st.customerSlots[id]; exists {
		st.mu.Unlock()
		return idx
	}
	idx := st.appendCustomerSlotLocked(normalizeBudget(initialBudget))
	st.customerSlots[id] = idx
	st.mu.Unlock()
	return idx
}

func (st *BudgetStore) LoadBudget(idx uint32) int64 {
	return st.loadOn(&st.budgets, idx)
}

func (st *BudgetStore) budgetSlotExists(idx uint32) bool {
	slice := st.budgets.Load()
	return idx < uint32(len(slice.data))
}

func (st *BudgetStore) CheckAndSpend(idx uint32, limit int64) bool {
	return st.checkAndSpendOn(&st.budgets, idx, limit)
}

func (st *BudgetStore) GetBudget(id CampaignID) int64 {
	st.mu.Lock()
	idx, exists := st.slots[id]
	if !exists {
		st.mu.Unlock()
		return 0
	}
	val := st.loadOn(&st.budgets, idx)
	st.mu.Unlock()
	return val
}

func (st *BudgetStore) CampaignSlot(id CampaignID) (uint32, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	idx, ok := st.slots[id]
	return idx, ok
}

func (st *BudgetStore) SetDailySpend(campaignIdx uint32, spent int64) {
	if spent < 0 {
		spent = 0
	}
	st.maybeRollDaily()
	st.addDailySpendLocked(campaignIdx, spent)
}

func (st *BudgetStore) SetBudget(id CampaignID, val int64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	idx, exists := st.slots[id]
	if !exists {
		idx = st.appendSlotLocked(normalizeBudget(val))
		st.slots[id] = idx
		return
	}
	slice := st.budgets.Load()
	if int(idx) >= len(slice.data) {
		return
	}
	atomic.StoreInt64(&slice.data[idx].Value, normalizeBudget(val))
}

func (st *BudgetStore) CustomerSlot(id CustomerID) (uint32, bool) {
	if id == 0 {
		return invalidCustomerBudgetIdx, false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	idx, ok := st.customerSlots[id]
	return idx, ok
}

func (st *BudgetStore) SetCustomerBudget(id CustomerID, val int64) {
	if id == 0 {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	idx, exists := st.customerSlots[id]
	if !exists {
		idx = st.appendCustomerSlotLocked(normalizeBudget(val))
		st.customerSlots[id] = idx
		return
	}
	slice := st.customerBudgets.Load()
	if int(idx) >= len(slice.data) {
		return
	}
	atomic.StoreInt64(&slice.data[idx].Value, normalizeBudget(val))
}

func (st *BudgetStore) appendSlotLocked(val int64) uint32 {
	currSlice := st.budgets.Load()
	idx := uint32(len(currSlice.data))

	newCap := cap(currSlice.data)
	if len(currSlice.data)+1 > newCap {
		if newCap == 0 {
			newCap = defaultBudgetCap
		} else {
			newCap *= 2
		}
	}

	newData := make([]AlignedBudget, len(currSlice.data)+1, newCap)
	copy(newData, currSlice.data)
	newData[idx] = AlignedBudget{Value: val}

	st.budgets.Store(&budgetSlice{data: newData})
	st.growDailyLocked(len(newData))
	return idx
}

func (st *BudgetStore) appendCustomerSlotLocked(val int64) uint32 {
	currSlice := st.customerBudgets.Load()
	idx := uint32(len(currSlice.data))

	newCap := cap(currSlice.data)
	if len(currSlice.data)+1 > newCap {
		if newCap == 0 {
			newCap = defaultBudgetCap
		} else {
			newCap *= 2
		}
	}

	newData := make([]AlignedBudget, len(currSlice.data)+1, newCap)
	copy(newData, currSlice.data)
	newData[idx] = AlignedBudget{Value: val}
	st.customerBudgets.Store(&budgetSlice{data: newData})
	return idx
}

func (st *BudgetStore) growDailyLocked(n int) {
	curr := st.dailySpent.Load()
	if len(curr.data) >= n {
		return
	}
	newData := make([]AlignedBudget, n, n*2)
	copy(newData, curr.data)
	st.dailySpent.Store(&budgetSlice{data: newData})
}

func normalizeBudget(val int64) int64 {
	if val < 0 {
		return 0
	}
	return val
}
