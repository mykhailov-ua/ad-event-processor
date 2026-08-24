package controlplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/dedup"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/dedupkey"

	"github.com/jackc/pgx/v5"
)

type RegionOutboxRelay struct {
	svc        *Service
	regionCode uint8
	outbox     *OutboxWorker
}

func NewRegionOutboxRelay(svc *Service) *RegionOutboxRelay {
	code := uint8(0)
	if svc != nil && svc.cfg != nil {
		code = svc.cfg.RegionCode
	}
	return &RegionOutboxRelay{
		svc:        svc,
		regionCode: code,
		outbox:     NewOutboxWorker(svc),
	}
}

func (r *RegionOutboxRelay) Start(ctx context.Context, interval time.Duration) {
	if r == nil || r.svc == nil || r.regionCode == 0 {
		return
	}
	if err := r.ProcessPending(ctx); err != nil {
		slog.Error("region outbox relay startup sync failed", "region", r.regionCode, "error", err)
	}
	slog.Info("region outbox relay starting", "region", r.regionCode, "interval", interval)

	pollBackoff := newOutboxPollBackoff()
	pollTimer := time.NewTimer(interval)
	defer pollTimer.Stop()

	recoveryTicker := time.NewTicker(interval * 5)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryTicker.C:
			r.reclaimStaleProcessing(ctx)
		case <-pollTimer.C:
			var processed int
			var err error
			if r.svc != nil {
				err = r.svc.withPgHigh(ctx, func(runCtx context.Context) error {
					var innerErr error
					processed, innerErr = r.ProcessPendingWithCount(runCtx, 500)
					return innerErr
				})
			} else {
				processed, err = r.ProcessPendingWithCount(ctx, 500)
			}
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("region outbox relay iteration failed", "region", r.regionCode, "error", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}
			pollTimer.Reset(pollBackoff.next(processed))
		}
	}
}

type regionDeliveryRow struct {
	deliveryID    int64
	outboxEventID int64
	eventType     string
	payload       []byte
	createdAt     time.Time
}

func (r *RegionOutboxRelay) reclaimStaleProcessing(ctx context.Context) {
	_, err := r.svc.GetPool().Exec(ctx, `
		UPDATE outbox_region_delivery
		SET status = 'PENDING', processing_started_at = NULL
		WHERE region_code = $1
		 AND status = 'PROCESSING'
		 AND processing_started_at IS NOT NULL
		 AND processing_started_at < NOW() - INTERVAL '1 minute'`, r.regionCode)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale region outbox deliveries", "region", r.regionCode, "error", err)
	}
}

func (r *RegionOutboxRelay) ProcessPending(ctx context.Context) error {
	_, err := r.ProcessPendingWithCount(ctx, 500)
	return err
}

func (r *RegionOutboxRelay) ProcessPendingWithCount(ctx context.Context, limit int32) (int, error) {
	if r == nil || r.svc == nil || r.regionCode == 0 {
		return 0, nil
	}

	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var rows []regionDeliveryRow
	err := pgx.BeginFunc(opCtx, r.svc.GetPool(), func(tx pgx.Tx) error {
		qrows, err := tx.Query(opCtx, `
			SELECT d.outbox_event_id, e.event_type, e.payload, e.created_at
			FROM outbox_region_delivery d
			JOIN outbox_events e ON e.id = d.outbox_event_id
			WHERE d.region_code = $1
			 AND d.status = 'PENDING'
			ORDER BY e.created_at ASC
			LIMIT $2
			FOR UPDATE OF d SKIP LOCKED`, r.regionCode, limit)
		if err != nil {
			return err
		}
		defer qrows.Close()

		var ids []int64
		for qrows.Next() {
			var row regionDeliveryRow
			if err := qrows.Scan(&row.outboxEventID, &row.eventType, &row.payload, &row.createdAt); err != nil {
				return err
			}
			row.deliveryID = row.outboxEventID
			rows = append(rows, row)
			ids = append(ids, row.outboxEventID)
		}
		if err := qrows.Err(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		_, err = tx.Exec(opCtx, `
			UPDATE outbox_region_delivery
			SET status = 'PROCESSING', processing_started_at = NOW()
			WHERE region_code = $1 AND outbox_event_id = ANY($2)`, r.regionCode, ids)
		return err
	})
	if err != nil || len(rows) == 0 {
		return 0, err
	}

	delivered := 0

	for i, row := range rows {
		if err := r.applyDelivery(opCtx, ctx, row); err != nil {
			slog.Warn("region outbox apply failed", "region", r.regionCode, "event_id", row.outboxEventID, "error", err)
			if revertErr := r.revertRegionDeliveriesPending(opCtx, rows[i:]); revertErr != nil {
				return delivered, errors.Join(
					fmt.Errorf("region delivery %d: %w", row.outboxEventID, err),
					fmt.Errorf("revert region deliveries: %w", revertErr),
				)
			}
			return delivered, fmt.Errorf("region delivery %d: %w", row.outboxEventID, err)
		}
		delivered++
	}
	return delivered, nil
}

func (r *RegionOutboxRelay) revertRegionDeliveriesPending(ctx context.Context, rows []regionDeliveryRow) error {
	if r == nil || r.svc == nil || len(rows) == 0 {
		return nil
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.outboxEventID
	}
	_, err := r.svc.GetPool().Exec(ctx, `
		UPDATE outbox_region_delivery
		SET status = 'PENDING', processing_started_at = NULL
		WHERE region_code = $1 AND outbox_event_id = ANY($2)`, r.regionCode, ids)
	return err
}

func (r *RegionOutboxRelay) regionAlreadyApplied(ctx context.Context, outboxEventID int64) (bool, error) {
	var already bool
	err := r.svc.GetPool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM region_apply_idempotency
			WHERE region_code = $1 AND outbox_event_id = $2
		)`, r.regionCode, outboxEventID).Scan(&already)
	return already, err
}

func (r *RegionOutboxRelay) markDelivered(ctx context.Context, row regionDeliveryRow) error {
	_, err := r.svc.GetPool().Exec(ctx, `
		UPDATE outbox_region_delivery
		SET status = 'DELIVERED', delivered_at = NOW(), processing_started_at = NULL
		WHERE region_code = $1 AND outbox_event_id = $2`, r.regionCode, row.outboxEventID)
	return err
}

func (r *RegionOutboxRelay) leasesEnabled() bool {
	return r != nil && r.svc != nil && r.svc.cfg != nil && r.svc.cfg.MultiRegionCell()
}

func (r *RegionOutboxRelay) relayLeaseWorker() *OperationLeaseWorker {
	if r.svc.leaseWorker != nil {
		return r.svc.leaseWorker
	}
	return NewOperationLeaseWorker(r.svc)
}

func (r *RegionOutboxRelay) applyDelivery(opCtx, ctx context.Context, row regionDeliveryRow) error {
	if r.leasesEnabled() {
		return r.applyDeliveryLeased(opCtx, ctx, row)
	}
	return r.applyDeliveryDirect(opCtx, ctx, row)
}

func (r *RegionOutboxRelay) applyDeliveryLeased(opCtx, ctx context.Context, row regionDeliveryRow) error {
	worker := r.relayLeaseWorker()
	bookReq := RelayDeliveryBookRequest(opCtx, r.svc, r.regionCode, row.outboxEventID, row.eventType, row.payload, 1)
	status, err := worker.EnsureBook(opCtx, bookReq)
	if err != nil {
		return fmt.Errorf("region delivery lease book event_id=%d: %w", row.outboxEventID, err)
	}
	if !status.QuorumMet {
		return fmt.Errorf("region delivery lease book event_id=%d: quorum not met", row.outboxEventID)
	}
	return worker.ExecuteOp(opCtx, bookReq.OpID, func(runCtx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		return r.applyDeliverySideEffects(runCtx, ctx, row, claim)
	})
}

func (r *RegionOutboxRelay) applyDeliveryDirect(opCtx, ctx context.Context, row regionDeliveryRow) error {
	adapter := r.svc.dedupAdapter(opCtx)
	var claim dedup.ClaimResult
	if adapter != nil {
		scope := adapter.RegionScope(dedupkey.RelaySourceID(r.regionCode), row.outboxEventID, row.outboxEventID)
		factorU := dedupkey.FactorU(dedupkey.CanonicalRelayPayload(row.outboxEventID, row.eventType, row.payload))
		var err error
		claim, err = adapter.ClaimConfirm(opCtx, scope, factorU)
		if err != nil {
			return err
		}
		if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
			return guardErr
		}
		if claim.Outcome == dedup.OutcomeAlreadyConfirmed {
			already, idemErr := r.regionAlreadyApplied(opCtx, row.outboxEventID)
			if idemErr != nil {
				return idemErr
			}
			if already {
				return r.markDelivered(opCtx, row)
			}
		}
		if r.svc != nil && len(r.svc.redisShards) > 0 && claim.DedupKey != "" {
			redisKey := dedupkey.RedisKey(claim.DedupKey)
			ok, nxErr := setNXOnAllShards(opCtx, r.svc.redisShards, redisKey, "1", 48*time.Hour)
			if nxErr != nil {
				return nxErr
			}
			if !ok && claim.Outcome == dedup.OutcomeConfirmed {
				already, idemErr := r.regionAlreadyApplied(opCtx, row.outboxEventID)
				if idemErr != nil {
					return idemErr
				}
				if already {
					return r.markDelivered(opCtx, row)
				}
			}
		}
	} else {
		already, err := r.regionAlreadyApplied(opCtx, row.outboxEventID)
		if err != nil {
			return err
		}
		if already {
			return r.markDelivered(opCtx, row)
		}
	}
	return r.applyDeliverySideEffects(opCtx, ctx, row, claim)
}

func (r *RegionOutboxRelay) applyDeliverySideEffects(opCtx, ctx context.Context, row regionDeliveryRow, claim dedup.ClaimResult) error {
	if claim.Outcome == dedup.OutcomeAlreadyConfirmed {
		already, idemErr := r.regionAlreadyApplied(opCtx, row.outboxEventID)
		if idemErr != nil {
			return idemErr
		}
		if already {
			return r.markDelivered(opCtx, row)
		}
	}
	if r.svc != nil && len(r.svc.redisShards) > 0 && claim.DedupKey != "" && claim.Outcome == dedup.OutcomeConfirmed {
		redisKey := dedupkey.RedisKey(claim.DedupKey)
		ok, nxErr := setNXOnAllShards(opCtx, r.svc.redisShards, redisKey, "1", 48*time.Hour)
		if nxErr != nil {
			return nxErr
		}
		if !ok {
			already, idemErr := r.regionAlreadyApplied(opCtx, row.outboxEventID)
			if idemErr != nil {
				return idemErr
			}
			if already {
				return r.markDelivered(opCtx, row)
			}
		}
	}

	ev := db.OutboxEvent{
		ID:        row.outboxEventID,
		EventType: row.eventType,
		Payload:   row.payload,
	}
	if err := r.outbox.handleOutboxEvent(opCtx, ctx, ev); err != nil {
		return err
	}

	err := pgx.BeginFunc(opCtx, r.svc.GetPool(), func(tx pgx.Tx) error {
		tag, err := tx.Exec(opCtx, `
			INSERT INTO region_apply_idempotency (region_code, outbox_event_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, r.regionCode, row.outboxEventID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		_, err = tx.Exec(opCtx, `
			UPDATE outbox_region_delivery
			SET status = 'DELIVERED', delivered_at = NOW(), processing_started_at = NULL
			WHERE region_code = $1 AND outbox_event_id = $2`, r.regionCode, row.outboxEventID)
		return err
	})
	if err != nil {
		return err
	}

	if !row.createdAt.IsZero() {
		lag := time.Since(row.createdAt).Seconds()
		if lag >= 0 {
			metrics.RegionOutboxDeliveryLag.Observe(lag)
		}
	}
	metrics.RegionOutboxDeliveredTotal.Inc()
	return nil
}
