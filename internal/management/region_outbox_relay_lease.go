package management

import (
	"context"
	"fmt"
	"time"

	"espx/internal/dedup"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/metrics"
	"espx/pkg/dedupkey"

	"github.com/jackc/pgx/v5"
)

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
	bookReq := RelayDeliveryBookRequest(r.svc, r.regionCode, row.outboxEventID, row.eventType, row.payload, 1)
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
	adapter := r.svc.dedupAdapter()
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
		if r.svc != nil && len(r.svc.rdbs) > 0 && claim.DedupKey != "" {
			redisKey := dedupkey.RedisKey(claim.DedupKey)
			ok, nxErr := r.svc.rdbs[0].SetNX(opCtx, redisKey, "1", 48*time.Hour).Result()
			if nxErr == nil && !ok && claim.Outcome == dedup.OutcomeConfirmed {
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
	if r.svc != nil && len(r.svc.rdbs) > 0 && claim.DedupKey != "" && claim.Outcome == dedup.OutcomeConfirmed {
		redisKey := dedupkey.RedisKey(claim.DedupKey)
		ok, nxErr := r.svc.rdbs[0].SetNX(opCtx, redisKey, "1", 48*time.Hour).Result()
		if nxErr == nil && !ok {
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
