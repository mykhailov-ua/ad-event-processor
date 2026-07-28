package management

import (
	"context"

	"espx/pkg/dedupkey"
)

func (s *Service) applyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error {
	if s == nil || s.globalSpend == nil {
		return nil
	}
	if !dedupkey.IsSpendSyncPayload(payload) {
		return nil
	}
	txns, err := dedupkey.DecodeSpendSyncPayload(payload)
	if err != nil {
		return err
	}
	return s.globalSpend.ApplyBatch(ctx, batchDedupKey, txns)
}
