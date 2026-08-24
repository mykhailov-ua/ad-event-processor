package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var PostSettlementMarkHook func(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error

type OutboxWorker struct {
	pool *pgxpool.Pool
	cfg  *config.Config

	apiMu sync.Mutex
	api   domain.PaymentSettlement

	settlementAlerter *SettlementFailedAlerter

	wg sync.WaitGroup
}

func (outboxWorker *OutboxWorker) SetSettlementFailedAlerter(alerter *SettlementFailedAlerter) {
	outboxWorker.settlementAlerter = alerter
}

func (outboxWorker *OutboxWorker) SetSettlementAPI(api domain.PaymentSettlement) {
	outboxWorker.apiMu.Lock()
	defer outboxWorker.apiMu.Unlock()
	outboxWorker.api = api
}

func NewOutboxWorker(pool *pgxpool.Pool, cfg *config.Config) *OutboxWorker {
	return &OutboxWorker{
		pool: pool,
		cfg:  cfg,
	}
}

type SettleBalancePayload struct {
	CustomerID           string `json:"customer_id"`
	AmountMicro          int64  `json:"amount_micro"`
	LedgerIdempotencyKey string `json:"ledger_idempotency_key"`
	PaymentIntentID      string `json:"payment_intent_id"`
	Provider             string `json:"provider"`
	ProviderRef          string `json:"provider_ref"`
}

func (outboxWorker *OutboxWorker) Start(ctx context.Context, interval time.Duration) {
	outboxWorker.wg.Add(1)
	defer outboxWorker.wg.Done()

	if err := outboxWorker.ensureSettlementClient(); err != nil {
		slog.Error("outbox worker failed to connect to management settlement server", "error", err)
	}

	slog.Info("payment outbox worker starting polling loop", "interval", interval)

	pollTimer := time.NewTimer(interval)
	defer pollTimer.Stop()

	recoveryTicker := time.NewTicker(interval * 5)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryTicker.C:
			outboxWorker.reclaimStaleProcessing(ctx)
		case <-pollTimer.C:
			outboxWorker.refreshOutboxPendingGauge(ctx)
			processed, err := outboxWorker.ProcessOutbox(ctx, 100)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Error("payment outbox processing iteration failed, retrying in 2s", "error", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}

			if processed > 0 {
				pollTimer.Reset(0)
				continue
			}

			pollTimer.Reset(interval)
		}
	}
}

func (outboxWorker *OutboxWorker) Wait() {
	outboxWorker.wg.Wait()
}

func (outboxWorker *OutboxWorker) refreshOutboxPendingGauge(ctx context.Context) {
	var pending int64
	err := outboxWorker.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment.payment_outbox
		WHERE status IN ('PENDING', 'PROCESSING')`).Scan(&pending)
	if err == nil {
		OutboxPending.Set(float64(pending))
	}
}

func (outboxWorker *OutboxWorker) reclaimStaleProcessing(ctx context.Context) {
	_, err := outboxWorker.pool.Exec(ctx, `
		UPDATE payment.payment_outbox
		SET status = 'PENDING', lease_until = NULL
		WHERE status = 'PROCESSING'
		 AND lease_until IS NOT NULL
		 AND lease_until < now()`)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale payment outbox outboxEventents", "error", err)
	}
}

func (outboxWorker *OutboxWorker) ProcessOutbox(ctx context.Context, limit int32) (int, error) {
	if err := outboxWorker.ensureSettlementClient(); err != nil {
		return 0, err
	}

	var outboxEventents []db.PaymentPaymentOutbox
	leaseDuration := 30 * time.Second
	leaseUntil := time.Now().Add(leaseDuration)

	err := pgx.BeginFunc(ctx, outboxWorker.pool, func(tx pgx.Tx) error {
		txQueries := db.New(tx)
		var err error
		outboxEventents, err = txQueries.GetPendingOutboxEventsForUpdate(ctx, limit)
		if err != nil || len(outboxEventents) == 0 {
			return err
		}

		ids := make([]int64, len(outboxEventents))
		for i := range outboxEventents {
			ids[i] = outboxEventents[i].ID
		}

		err = txQueries.LeaseOutboxEvents(ctx, db.LeaseOutboxEventsParams{
			Column1:    ids,
			LeaseUntil: pgtype.Timestamptz{Time: leaseUntil, Valid: true},
		})
		return err
	})

	if err != nil || len(outboxEventents) == 0 {
		return 0, err
	}

	successCount := 0
	var batchErrs []error
	for i := range outboxEventents {
		outboxEvent := outboxEventents[i]
		if err := outboxWorker.handleOutboxEvent(ctx, outboxEvent); err != nil {
			slog.Error("failed to handle outbox outboxEventent", "id", outboxEvent.ID, "error", err)
			SettlementErrorsTotal.Inc()
			outboxWorker.markOutboxEventRetryable(ctx, outboxEvent, err)
			batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", outboxEvent.ID, err))
			continue
		}
		if PostSettlementMarkHook != nil {
			if hookErr := PostSettlementMarkHook(ctx, outboxEvent); hookErr != nil {
				slog.Error("post-settlement hook failed", "id", outboxEvent.ID, "error", hookErr)
				SettlementErrorsTotal.Inc()
				outboxWorker.markOutboxEventRetryable(ctx, outboxEvent, hookErr)
				batchErrs = append(batchErrs, fmt.Errorf("outbox event %d post-settlement hook: %w", outboxEvent.ID, hookErr))
				continue
			}
		}
		if err := outboxWorker.markOutboxProcessedWithRetry(ctx, outboxEvent.ID); err != nil {
			slog.Error("failed to mark outbox outboxEventent processed", "id", outboxEvent.ID, "error", err)
			batchErrs = append(batchErrs, fmt.Errorf("mark outbox processed %d: %w", outboxEvent.ID, err))
			continue
		}
		successCount++
	}

	outboxWorker.refreshOutboxPendingGauge(ctx)
	if len(batchErrs) > 0 {
		return successCount, errors.Join(batchErrs...)
	}
	return successCount, nil
}

func (outboxWorker *OutboxWorker) ensureSettlementClient() error {
	outboxWorker.apiMu.Lock()
	defer outboxWorker.apiMu.Unlock()

	if outboxWorker.api != nil {
		return nil
	}
	return fmt.Errorf("settlement API not injected")
}

func (outboxWorker *OutboxWorker) getSettlementAPI() domain.PaymentSettlement {
	outboxWorker.apiMu.Lock()
	defer outboxWorker.apiMu.Unlock()
	return outboxWorker.api
}

func (outboxWorker *OutboxWorker) markOutboxProcessedWithRetry(ctx context.Context, outboxID int64) error {
	var lastErr error
	for attempt := range 3 {
		lastErr = db.New(outboxWorker.pool).MarkOutboxEventProcessed(ctx, outboxID)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return lastErr
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return lastErr
}

func (outboxWorker *OutboxWorker) markOutboxEventRetryable(ctx context.Context, outboxEvent db.PaymentPaymentOutbox, cause error) {
	var lastErrText pgtype.Text
	lastErrText.String = cause.Error()
	lastErrText.Valid = true

	isFatal := domain.IsSettlementNotFound(cause)

	maxAttempts := int32(outboxWorker.cfg.MaxRetries)
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	permanent := isFatal || outboxEvent.Attempts+1 >= maxAttempts

	innerErr := pgx.BeginFunc(ctx, outboxWorker.pool, func(tx pgx.Tx) error {
		txQueries := db.New(tx)
		if isFatal {
			if err := txQueries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
				ID:        outboxEvent.ID,
				Attempts:  0,
				LastError: lastErrText,
			}); err != nil {
				return err
			}

			if outboxEvent.EventType == "SETTLE_BALANCE" {
				payload, err := decodeOutboxPayload[SettleBalancePayload](outboxEvent, "settle balance")
				if err != nil {
					slog.Warn("settlement failed outbox payload decode", "error", err, "outbox_id", outboxEvent.ID)
				} else {
					intentUUID, parseErr := uuid.Parse(payload.PaymentIntentID)
					if parseErr != nil {
						slog.Warn("settlement failed outbox invalid intent id", "error", parseErr, "outbox_id", outboxEvent.ID)
					} else if _, updateErr := txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
						ID:          pgtype.UUID{Bytes: intentUUID, Valid: true},
						Status:      db.PaymentPaymentIntentStatusSETTLEMENTFAILED,
						ProviderRef: pgtype.Text{String: payload.ProviderRef, Valid: true},
					}); updateErr != nil {
						return fmt.Errorf("mark intent settlement failed: %w", updateErr)
					}
				}
			}
		} else {
			if err := txQueries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
				ID:        outboxEvent.ID,
				Attempts:  maxAttempts,
				LastError: lastErrText,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if innerErr != nil {
		slog.Error("failed to update outbox outboxEventent failure status", "id", outboxEvent.ID, "error", innerErr)
		return
	}
	if permanent && outboxWorker.settlementAlerter != nil {
		outboxWorker.settlementAlerter.AlertPermanentFailure(ctx, outboxEvent, cause)
	}
}

func (outboxWorker *OutboxWorker) handleOutboxEvent(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error {
	switch outboxEvent.EventType {
	case "SETTLE_BALANCE":
		return outboxWorker.handleSettleBalance(ctx, outboxEvent)
	case OutboxEventReverseBalance:
		return outboxWorker.handleReverseBalance(ctx, outboxEvent)
	case OutboxEventApplyChargeback:
		return outboxWorker.handleApplyChargeback(ctx, outboxEvent)
	case OutboxEventReverseChargeback:
		return outboxWorker.handleReverseChargeback(ctx, outboxEvent)
	default:
		slog.Warn("skipping unrecognized payment outbox outboxEventent type", "type", outboxEvent.EventType)
		return nil
	}
}

func (outboxWorker *OutboxWorker) handleSettleBalance(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error {
	return applySettlementOutbox(ctx, outboxWorker, outboxEvent, "settle balance",
		func(api domain.PaymentSettlement, customerID, paymentIntentID uuid.UUID, payload SettleBalancePayload) error {
			_, _, err := api.ApplyPaymentCredit(ctx, customerID, payload.AmountMicro, payload.LedgerIdempotencyKey, paymentIntentID, payload.Provider, payload.ProviderRef)
			if err != nil {
				return fmt.Errorf("management SettlementService call failed: %w", err)
			}
			return nil
		},
	)
}

func (outboxWorker *OutboxWorker) handleReverseBalance(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error {
	return applySettlementOutbox(ctx, outboxWorker, outboxEvent, "reverse balance",
		func(api domain.PaymentSettlement, customerID, paymentIntentID uuid.UUID, payload ReverseBalancePayload) error {
			_, _, err := api.ApplyPaymentRefund(ctx, customerID, payload.AmountMicro, payload.LedgerIdempotencyKey, paymentIntentID, payload.Provider, payload.ProviderRefundID)
			if err != nil {
				return fmt.Errorf("management SettlementService refund call failed: %w", err)
			}
			return nil
		},
	)
}

func (outboxWorker *OutboxWorker) handleApplyChargeback(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error {
	return applySettlementOutbox(ctx, outboxWorker, outboxEvent, "apply chargeback",
		func(api domain.PaymentSettlement, customerID, paymentIntentID uuid.UUID, payload ApplyChargebackPayload) error {
			_, _, err := api.ApplyPaymentChargeback(ctx, customerID, payload.AmountMicro, payload.LedgerIdempotencyKey, paymentIntentID, payload.Provider, payload.ProviderDisputeID)
			if err != nil {
				return fmt.Errorf("management SettlementService chargeback call failed: %w", err)
			}
			return nil
		},
	)
}

func (outboxWorker *OutboxWorker) handleReverseChargeback(ctx context.Context, outboxEvent db.PaymentPaymentOutbox) error {
	return applySettlementOutbox(ctx, outboxWorker, outboxEvent, "reverse chargeback",
		func(api domain.PaymentSettlement, customerID, paymentIntentID uuid.UUID, payload ReverseChargebackPayload) error {
			_, _, err := api.ApplyPaymentChargebackReversal(ctx, customerID, payload.AmountMicro, payload.LedgerIdempotencyKey, paymentIntentID, payload.Provider, payload.ProviderDisputeID)
			if err != nil {
				return fmt.Errorf("management SettlementService chargeback reversal call failed: %w", err)
			}
			return nil
		},
	)
}
