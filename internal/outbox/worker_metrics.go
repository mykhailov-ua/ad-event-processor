package outbox

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/metrics"
	db "ad-event-processor/internal/domain/db"
)

func (w *Worker) ApplyQuotaRepair(ctx context.Context, eventID int64, payload []byte) error {
	if w == nil || w.host == nil {
		return nil
	}
	return w.host.ApplyQuotaRepair(ctx, eventID, payload)
}

func (w *Worker) ApplyReconciliationAdjust(ctx context.Context, eventID int64, payload []byte) error {
	if w == nil || w.host == nil {
		return nil
	}
	return w.host.ApplyReconciliationAdjust(ctx, eventID, payload)
}

func (w *Worker) recordOutboxLagMetrics(ctx context.Context) {
	if w.host == nil || w.host.Pool() == nil {
		return
	}
	opCtx, cancel := workerContext(ctx, WorkerTimeout)
	defer cancel()

	var pending int64
	var oldestSeconds float64
	err := w.host.Pool().QueryRow(opCtx, `
		SELECT COUNT(*)::bigint,
		 COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0)::float8
		FROM outbox_events
		WHERE status = 'PENDING'`).Scan(&pending, & oldestSeconds)
	if err != nil {
		if ctx.Err() != nil || database.IsShutdownError(err) {
			return
		}
		return
	}
	w.RecordOutboxLagFromValues(ctx, pending, oldestSeconds)
}

func (w *Worker) RecordOutboxLagFromValues(ctx context.Context, pending int64, oldestSeconds float64) {
	metrics.SetControlOutboxQueueMetrics(pending, oldestSeconds)
	alerter := w.host.OutboxAlerter()
	if alerter != nil && pending > 0 {
		threshold := float64(alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			alerter.AlertOutboxStuck(ctx, pending, oldestSeconds)
		}
	}
}

func (w *Worker) HandleUpdateSettings(opCtx context.Context, eventID int64, payload []byte) error {
	return w.handleUpdateSettings(opCtx, eventID, payload)
}

func (w *Worker) HandleCreateCampaign(ctx context.Context, payload []byte) error {
	return w.handleCreateCampaign(ctx, payload)
}

func (w *Worker) HandleUpdateCampaignSchedule(ctx context.Context, payload []byte) error {
	return w.handleUpdateCampaignSchedule(ctx, payload)
}

func (w *Worker) ApplyBlacklistPayload(ctx context.Context, p BlacklistPayload, queuedAt time.Time) error {
	return w.applyBlacklistPayload(ctx, p, queuedAt)
}

func (w *Worker) ApplyBlacklistPayloadsBatch(ctx context.Context, events []db.OutboxEvent) error {
	return w.applyBlacklistPayloadsBatch(ctx, events)
}

func (w *Worker) HandleLanderPublished(ctx context.Context, payload []byte) error {
	return w.handleLanderPublished(ctx, payload)
}
