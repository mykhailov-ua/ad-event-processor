package ingestion

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
)

type LocalQuantaDeps struct {
	Ledger    *LocalQuantaLedger
	Strict    *LocalQuantaStrict
	Refill    *QuotaRefillWorker
	Publisher *BudgetDeltaPublisher
	Stream    *LocalQuantaStreamPublisher
}

func (f *UnifiedFilter) SetLocalQuantaDeps(deps LocalQuantaDeps) {
	f.localQuantaLedger = deps.Ledger
	f.localQuantaStrict = deps.Strict
	f.localQuantaRefill = deps.Refill
	f.localQuantaPublisher = deps.Publisher
	f.localQuantaStream = deps.Stream
	if deps.Stream != nil {
		f.localClickIdem = deps.Stream.IdemCache()
	}
}

func (f *UnifiedFilter) SetLocalQuantaMode(mode string) {
	f.localQuotaMode = mode
	if f.localQuantaLedger != nil {
		f.localQuantaLedger.SetMode(mode)
	}
}

func (f *UnifiedFilter) localQuantaActive() bool {
	return f.localQuotaMode == "shadow" || f.localQuotaMode == "live"
}

func (f *UnifiedFilter) localQuantaEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	if f.localQuantaLedger == nil || !f.localQuantaActive() {
		return false
	}
	if f.quotaEnabledAny != oneAny {
		return false
	}
	if f.localQuantaStrict != nil && f.localQuantaStrict.IsStrict(evt.CampaignID) {
		return false
	}
	if !f.fastPathEnabled.Load() || f.needsFullLuaPath(evt, campInfo) {
		return false
	}
	if evt.Type != "impression" && evt.Type != "click" {
		return false
	}
	return true
}

func (f *UnifiedFilter) localQuantaFullSkipEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	if f.localQuotaMode != "live" || f.localQuantaStream == nil {
		return false
	}
	if !f.localQuantaEligible(evt, campInfo) {
		return false
	}
	return true
}

// acceptLocalQuantaFullSkip skips sync Redis EVALSHA: placement and fraud blacklist are checked in Go
// before local quanta; ingress RPD via EntitlementsFilter; click dedup via localClickIdem
// (async SET NX in stream worker).

func (f *UnifiedFilter) checkLocalQuanta(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amountMicro int64,
) (handled bool, err error) {
	if !f.localQuantaEligible(evt, campInfo) {
		return false, nil
	}

	amount := amountMicro
	if amount <= 0 {
		if evt.Type == "impression" {
			amount = f.impressionAmountMicro
		} else {
			amount = f.clickAmountMicro
		}
	}

	if f.localQuotaMode == "shadow" {
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		localOK := f.localQuantaLedger.TrySpendDebit(evt.CampaignID, subSlot, amount)
		if localOK {
			f.publishLocalDelta(evt.CampaignID, amount)
		}
		return false, nil
	}

	if campInfo.FreqLimit > 0 && evt.UserID != "" {
		exceeded, err := f.checkFreqLimitGo(evt, campInfo)
		if err != nil {
			return true, err
		}
		if exceeded {
			return true, ErrFreqLimitExceeded
		}
	}

	subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
	if !f.localQuantaLedger.TrySpendDebit(evt.CampaignID, subSlot, amount) {
		if f.localQuantaRefill != nil {
			f.localQuantaRefill.Signal(evt.CampaignID)
		}
		return false, nil
	}

	metrics.LocalQuotaSpendTotal.Inc()
	f.publishLocalDelta(evt.CampaignID, amount)

	if f.localQuantaFullSkipEligible(evt, campInfo) {
		metrics.LocalQuotaFullSkipEligibleTotal.Inc()
		err := f.acceptLocalQuantaFullSkip(ctx, evt, campInfo, amount, subSlot)
		return true, err
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amount)
		return true, err
	}
	rdb := f.rdbs[shard%len(f.rdbs)]

	debitAny := f.clickAmountMicroAny
	if evt.Type == "impression" {
		debitAny = f.impressionAmountMicroAny
	}

	prevSkip := f.skipBudgetDebitAny
	f.skipBudgetDebitAny = oneAny
	fastScratch := budgetFastScratchPool.Get().(*budgetFastScratch)
	err = f.runBudgetFastLua(ctx, evt, campInfo, debitAny, rdb, shard, fastScratch)
	f.skipBudgetDebitAny = prevSkip
	budgetFastScratchPool.Put(fastScratch)

	if err != nil {
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amount)
		return true, err
	}
	return true, nil
}

func (f *UnifiedFilter) rollbackLocalQuantaSpend(campaignID uuid.UUID, subSlot int, amountMicro int64) {
	if f.localQuantaLedger != nil && amountMicro > 0 {
		f.localQuantaLedger.RefundDebit(campaignID, subSlot, amountMicro)
	}
	if f.localQuantaPublisher != nil && amountMicro > 0 {
		f.localQuantaPublisher.PublishReturn(campaignID, amountMicro)
	}
}

func (f *UnifiedFilter) acceptLocalQuantaFullSkip(ctx context.Context, evt *domain.Event, campInfo *domain.Campaign, amountMicro int64, subSlot int) error {
	if f.localClickIdem != nil && !f.localClickIdem.TryClaim(evt.ClickID) {
		metrics.FilterLuaBranchTotal.WithLabelValues("duplicate").Inc()
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return ErrDuplicateEvent
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		if f.localClickIdem != nil {
			f.localClickIdem.Release(evt.ClickID)
		}
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return err
	}

	if !f.localQuantaStream.Enqueue(shard, evt, campInfo, amountMicro) {
		if f.localClickIdem != nil {
			f.localClickIdem.Release(evt.ClickID)
		}
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return ErrShardUnavailable
	}

	metrics.LocalQuotaFullSkipTotal.Inc()
	metrics.RedisLuaSkippedTotal.Inc()
	metrics.EventsProcessed.Inc()
	telemetry.RecordAccepted()
	return nil
}

func (f *UnifiedFilter) publishLocalDelta(campaignID uuid.UUID, amountMicro int64) {
	if f.localQuantaPublisher != nil {
		f.localQuantaPublisher.Publish(campaignID, amountMicro)
	}
}

func (f *UnifiedFilter) RecordShadowLuaOutcome(campaignID uuid.UUID, luaBudgetExhausted bool) {
	if f.localQuotaMode != "shadow" || f.localQuantaLedger == nil {
		return
	}
	localHad := f.localQuantaLedger.Remaining(campaignID) >= 0 && f.localQuantaLedger.HasCredit(campaignID)
	if localHad && luaBudgetExhausted {
		metrics.LocalQuotaShadowDiffTotal.Inc()
	}
}

func (f *UnifiedFilter) UpdateStrictFromRedis(campaignID uuid.UUID, redisRemaining int64) {
	if f.localQuantaStrict != nil {
		f.localQuantaStrict.UpdateFromRedisRemaining(campaignID, redisRemaining)
	}
}
