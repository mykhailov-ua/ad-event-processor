package rtb

import (
	"sync/atomic"
	"time"
)

const invalidCustomerBudgetIdx uint32 = ^uint32(0)

func (st *BudgetStore) CheckAndSpendAll(campaignIdx, customerIdx uint32, price, dailyLimit int64) bool {
	if dailyLimit > 0 {
		st.maybeRollDaily()
		if st.loadDailyHeadroom(campaignIdx, dailyLimit) < price {
			return false
		}
	}

	if !st.checkAndSpendOn(&st.budgets, campaignIdx, price) {
		return false
	}

	if customerIdx != invalidCustomerBudgetIdx {
		if !st.checkAndSpendOn(&st.customerBudgets, customerIdx, price) {
			st.creditOn(&st.budgets, campaignIdx, price)
			return false
		}
	}

	if dailyLimit > 0 {
		if !st.checkAndAddDailySpend(campaignIdx, price, dailyLimit) {
			if customerIdx != invalidCustomerBudgetIdx {
				st.creditOn(&st.customerBudgets, customerIdx, price)
			}
			st.creditOn(&st.budgets, campaignIdx, price)
			return false
		}
	}

	return true
}

func (st *BudgetStore) loadDailyHeadroom(campaignIdx uint32, dailyLimit int64) int64 {
	if dailyLimit <= 0 {
		return dailyLimit
	}
	spent := st.loadOn(&st.dailySpent, campaignIdx)
	return dailyLimit - spent
}

func (st *BudgetStore) LoadCustomerBudget(customerIdx uint32) int64 {
	if customerIdx == invalidCustomerBudgetIdx {
		return 0
	}
	return st.loadOn(&st.customerBudgets, customerIdx)
}

func (st *BudgetStore) checkAndSpendOn(holder *atomic.Pointer[budgetSlice], idx uint32, price int64) bool {
	slice := holder.Load()
	if idx >= uint32(len(slice.data)) {
		return false
	}
	ptr := &slice.data[idx].Value
	for {
		curr := atomic.LoadInt64(ptr)
		if curr < price {
			return false
		}
		if atomic.CompareAndSwapInt64(ptr, curr, curr-price) {
			return true
		}
	}
}

func (st *BudgetStore) creditOn(holder *atomic.Pointer[budgetSlice], idx uint32, price int64) {
	slice := holder.Load()
	if idx >= uint32(len(slice.data)) {
		return
	}
	ptr := &slice.data[idx].Value
	for {
		curr := atomic.LoadInt64(ptr)
		if atomic.CompareAndSwapInt64(ptr, curr, curr+price) {
			return
		}
	}
}

func (st *BudgetStore) checkAndAddDailySpend(idx uint32, price, dailyLimit int64) bool {
	slice := st.dailySpent.Load()
	if idx >= uint32(len(slice.data)) {
		return false
	}
	ptr := &slice.data[idx].Value
	for {
		curr := atomic.LoadInt64(ptr)
		if curr+price > dailyLimit {
			return false
		}
		if atomic.CompareAndSwapInt64(ptr, curr, curr+price) {
			return true
		}
	}
}

func (st *BudgetStore) addDailySpendLocked(idx uint32, spent int64) {
	slice := st.dailySpent.Load()
	if idx >= uint32(len(slice.data)) {
		return
	}
	atomic.StoreInt64(&slice.data[idx].Value, spent)
}

func (st *BudgetStore) loadOn(holder *atomic.Pointer[budgetSlice], idx uint32) int64 {
	slice := holder.Load()
	if idx >= uint32(len(slice.data)) {
		return 0
	}
	return atomic.LoadInt64(&slice.data[idx].Value)
}

func (st *BudgetStore) maybeRollDaily() {
	day := currentDayEpochUTC()
	if st.dailyEpoch.Load() == day {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.dailyEpoch.Load() == day {
		return
	}
	curr := st.dailySpent.Load()
	if len(curr.data) > 0 {
		cleared := make([]AlignedBudget, len(curr.data))
		st.dailySpent.Store(&budgetSlice{data: cleared})
	}
	st.dailyEpoch.Store(day)
}

func currentDayEpochUTC() uint32 {
	now := time.Now().UTC()
	return uint32(now.Year()*10000 + int(now.Month())*100 + now.Day())
}
