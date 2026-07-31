package controlplane

import (
	"context"

	"espx/internal/database"
	"espx/internal/metrics"
)

func (worker *OutboxWorker) recordOutboxLagMetrics(ctx context.Context) {
	if worker.svc == nil || worker.svc.GetPool() == nil {
		return
	}
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var pending int64
	var oldestSeconds float64
	err := worker.svc.GetPool().QueryRow(opCtx, `
		SELECT COUNT(*)::bigint,
		       COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0)::float8
		FROM outbox_events
		WHERE status = 'PENDING'`).Scan(&pending, &oldestSeconds)
	if err != nil {
		if ctx.Err() != nil || database.IsShutdownError(err) {
			return
		}
		return
	}
	metrics.ManagementOutboxPendingTotal.Set(float64(pending))
	metrics.ManagementOutboxOldestPendingSeconds.Set(oldestSeconds)

	if worker.svc != nil && worker.svc.alerter != nil && pending > 0 {
		threshold := float64(worker.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			worker.svc.alerter.AlertOutboxStuck(pending, oldestSeconds)
		}
	}
}

func (worker *OutboxWorker) recordOutboxLagFromValues(pending int64, oldestSeconds float64) {
	metrics.ManagementOutboxPendingTotal.Set(float64(pending))
	metrics.ManagementOutboxOldestPendingSeconds.Set(oldestSeconds)
	if worker.svc != nil && worker.svc.alerter != nil && pending > 0 {
		threshold := float64(worker.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			worker.svc.alerter.AlertOutboxStuck(pending, oldestSeconds)
		}
	}
}
