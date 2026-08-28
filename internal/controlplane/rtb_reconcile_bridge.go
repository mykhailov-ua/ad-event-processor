package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/reconciliation"
)

func (s *Service) RtbReconcileCHStats(ctx context.Context, requestID string, window time.Duration) (reconciliation.RtbReconcileCHStats, bool) {
	return reconciliation.RTBCHStats(ctx, s, requestID, window)
}
