package ingestion

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/bidshard/ad-event-processor/internal/domain"
)

//go:embed budget-rollback.lua
var budgetRollbackLua string

func (f *UnifiedFilter) RollbackRedisDebit(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount int64,
) error {
	if f == nil || evt == nil || campInfo == nil {
		return nil
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		return fmt.Errorf("rollback: failed to resolve shard: %w", err)
	}
	rdb := f.rdbs[shard%len(f.rdbs)]
	if rdb == nil {
		return fmt.Errorf("rollback: redis client is nil for shard %d", shard)
	}

	budgetSourceKey := campInfo.BudgetCampaignKey
	if f.quotaEnabledAny == oneAny {
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		var buf []byte
		buf = appendBudgetQuotaKey(buf, evt.CampaignID, subSlot)
		budgetSourceKey = unsafeString(buf)
	}

	var idemBuf []byte
	idemBuf = append(idemBuf, "idempotency:click:"...)
	idemBuf = append(idemBuf, evt.ClickID...)
	idempotencyKey := unsafeString(idemBuf)

	keys := []string{
		budgetSourceKey,
		idempotencyKey,
		campInfo.CampaignSyncKey,
		campInfo.CustomerSyncKey,
		dirtyCampaignsKeyVal.s,
		dirtyCustomersKeyVal.s,
	}

	args := []any{
		amount,
		campInfo.ID.String(),
		campInfo.CustomerID.String(),
	}

	err = rdb.EvalSha(ctx, f.rollbackScriptHash, keys, args...).Err()
	if err != nil && isNoScriptErr(err) {
		err = rdb.Eval(ctx, budgetRollbackLua, keys, args...).Err()
	}

	if err != nil {
		slog.Error("failed to rollback redis debit",
			"campaign_id", evt.CampaignID,
			"click_id", evt.ClickID,
			"amount", amount,
			"error", err,
		)
		return err
	}

	slog.Info("successfully rolled back redis debit",
		"campaign_id", evt.CampaignID,
		"click_id", evt.ClickID,
		"amount", amount,
	)
	return nil
}

func (f *UnifiedFilter) RollbackDebit(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount int64,
	isLocalQuanta bool,
) {
	if f == nil || evt == nil || campInfo == nil {
		return
	}
	if isLocalQuanta {
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amount)
		if f.localClickIdem != nil {
			f.localClickIdem.Release(evt.ClickID)
		}
	} else {
		_ = f.RollbackRedisDebit(ctx, evt, campInfo, amount)
	}
}
