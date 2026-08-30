package rtb

import "sync/atomic"

// BudgetSpendMirror receives post-CAS spend notifications from RunAuction (live path only).
// RtbBudgetMirrorWriter implements async Redis DECRBY when RTB_BUDGET_AUTHORITY=rtb so in-process
// BudgetStore stays authoritative while Redis mirrors for reconcile; authority=redis skips RTB CAS entirely.
type BudgetSpendMirror interface {
	RecordSpend(campaignID CampaignID, budgetIdx uint32, priceMicro int64)
}

var globalBudgetSpendMirror atomic.Pointer[BudgetSpendMirror]

func SetBudgetSpendMirror(m BudgetSpendMirror) {
	if m == nil {
		globalBudgetSpendMirror.Store(nil)
		return
	}
	globalBudgetSpendMirror.Store(&m)
}

// recordBudgetSpendMirror is best-effort; nil mirror is normal when authority=redis or shadow eval.
func recordBudgetSpendMirror(campaignID CampaignID, budgetIdx uint32, priceMicro int64) {
	ptr := globalBudgetSpendMirror.Load()
	if ptr == nil || *ptr == nil {
		return
	}
	(*ptr).RecordSpend(campaignID, budgetIdx, priceMicro)
}
