package ingestion

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

func (c *StreamConsumer) splitStoreBatch(ctx context.Context, batch []*domain.Event, msgIDs []string, baseIdx int) (successIdx, failIdx []int) {
	if len(batch) == 0 {
		return nil, nil
	}

	storeCtx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	if len(msgIDs) > 0 {
		token := fmt.Sprintf("%s_%s_%d", msgIDs[0], msgIDs[len(msgIDs)-1], len(msgIDs))
		storeCtx = context.WithValue(storeCtx, domain.DeduplicationTokenKey, token)
	}
	err := c.store.StoreBatch(storeCtx, batch)
	cancel()

	if err == nil {
		successIdx = make([]int, len(batch))
		for i := range batch {
			successIdx[i] = baseIdx + i
		}
		return successIdx, nil
	}

	if isRetriableStoreError(err) {
		for i := range batch {
			failIdx = append(failIdx, baseIdx+i)
		}
		return nil, failIdx
	}

	if len(batch) == 1 {
		metrics.CHSingleRowInsertsTotal.Inc()
		return nil, []int{baseIdx}
	}

	mid := len(batch) / 2
	leftOK, leftFail := c.splitStoreBatch(ctx, batch[:mid], msgIDs[:mid], baseIdx)
	rightOK, rightFail := c.splitStoreBatch(ctx, batch[mid:], msgIDs[mid:], baseIdx+mid)
	return append(leftOK, rightOK...), append(leftFail, rightFail...)
}
