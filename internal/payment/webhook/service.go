package webhook

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/payment/checkout"
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/internal/payment/settlement"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewService(pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{pool: pool, cfg: cfg}
}

func isValidTransition(oldStatus, newStatus db.PaymentPaymentIntentStatus) bool {
	if oldStatus == newStatus {
		return true
	}
	switch oldStatus {
	case db.PaymentPaymentIntentStatusCREATED:
		return true
	case db.PaymentPaymentIntentStatusPENDINGPROVIDER:
		return newStatus != db.PaymentPaymentIntentStatusCREATED
	case db.PaymentPaymentIntentStatusPROCESSING:
		return newStatus != db.PaymentPaymentIntentStatusCREATED &&
			newStatus != db.PaymentPaymentIntentStatusPENDINGPROVIDER
	case db.PaymentPaymentIntentStatusSUCCEEDED:
		return newStatus == db.PaymentPaymentIntentStatusREFUNDED ||
			newStatus == db.PaymentPaymentIntentStatusDISPUTED
	case db.PaymentPaymentIntentStatusREFUNDED:
		return newStatus == db.PaymentPaymentIntentStatusDISPUTED
	case db.PaymentPaymentIntentStatusDISPUTED:
		return newStatus == db.PaymentPaymentIntentStatusSUCCEEDED
	case db.PaymentPaymentIntentStatusFAILED,
		db.PaymentPaymentIntentStatusCANCELLED,
		db.PaymentPaymentIntentStatusSETTLEMENTFAILED:
		return false
	default:
		return true
	}
}

func ledgerIdempotencyKey(intentID uuid.UUID) string {
	return "payment:" + intentID.String()
}

func (s *Service) ProcessStripeWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, rawEvent string) error {
	h := sha256.New()
	h.Write(payload)
	payloadHash := h.Sum(nil)

	redactedBytes, err := coldpath.RedactStripeWebhookPayload(payload)
	if err != nil {
		return fmt.Errorf("redact stripe webhook payload: %w", err)
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		txQueries := db.New(tx)

		_, err := txQueries.GetWebhookEvent(ctx, db.GetWebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: eventID,
		})
		if err == nil {
			slog.Info("webhook event already processed", "event_id", eventID)
			WebhookEventsTotal.WithLabelValues("duplicate").Inc()
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		_, err = txQueries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: eventID,
			EventType:       eventType,
			PayloadHash:     payloadHash,
			PayloadRedacted: redactedBytes,
			Status:          db.PaymentWebhookEventStatusRECEIVED,
			ErrorMessage:    pgtype.Text{},
		})
		if err != nil {
			if coldpath.IsUniqueViolation(err) {
				slog.Info("webhook event deduplicated by unique constraint", "event_id", eventID)
				WebhookEventsTotal.WithLabelValues("duplicate").Inc()
				return nil
			}
			return err
		}

		var intent db.PaymentPaymentIntent
		err = tx.QueryRow(ctx, `
			SELECT id, customer_id, amount_micro, currency, status, provider, provider_ref, idempotency_key, metadata, created_at, updated_at
			FROM payment.payment_intents
			WHERE provider = 'stripe' AND provider_ref = $1
			FOR UPDATE`, providerRef).Scan(
			&intent.ID, &intent.CustomerID, &intent.AmountMicro, &intent.Currency, &intent.Status, &intent.Provider, &intent.ProviderRef, &intent.IdempotencyKey, &intent.Metadata, &intent.CreatedAt, &intent.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Warn("received stripe event for unknown provider_ref", "provider_ref", providerRef)
				return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "unknown provider_ref")
			}
			return err
		}

		var targetStatus db.PaymentPaymentIntentStatus
		switch eventType {
		case "payment_intent.succeeded":
			targetStatus = db.PaymentPaymentIntentStatusSUCCEEDED
		case "payment_intent.payment_failed":
			targetStatus = db.PaymentPaymentIntentStatusFAILED
		case "payment_intent.canceled":
			targetStatus = db.PaymentPaymentIntentStatusCANCELLED
		default:
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
		}

		if targetStatus == db.PaymentPaymentIntentStatusSUCCEEDED && amountMicro <= 0 {
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "zero or negative amount")
		}

		if amountMicro != intent.AmountMicro {
			slog.Warn("webhook amount mismatch", "intent_id", uuid.UUID(intent.ID.Bytes), "intent_amount", intent.AmountMicro, "webhook_amount", amountMicro)
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "amount mismatch")
		}

		if !isValidTransition(intent.Status, targetStatus) {
			slog.Warn("invalid state transition skipped", "intent_id", uuid.UUID(intent.ID.Bytes), "from", intent.Status, "to", targetStatus)
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED,
				fmt.Sprintf("invalid transition from %s to %s", intent.Status, targetStatus))
		}

		alreadySettled := intent.Status == db.PaymentPaymentIntentStatusSUCCEEDED

		_, err = txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
			ID:          intent.ID,
			Status:      targetStatus,
			ProviderRef: pgtype.Text{String: providerRef, Valid: true},
		})
		if err != nil {
			return err
		}

		if targetStatus == db.PaymentPaymentIntentStatusSUCCEEDED && !alreadySettled {
			intentUUID := uuid.UUID(intent.ID.Bytes)
			payloadJSON, err := settlement.MarshalSettleBalanceOutbox(uuid.UUID(intent.CustomerID.Bytes), intent.AmountMicro, intentUUID, "stripe", providerRef)
			if err != nil {
				return fmt.Errorf("marshal settle balance outbox payload: %w", err)
			}
			_, err = txQueries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "SETTLE_BALANCE",
				Payload:   payloadJSON,
			})
			if err != nil {
				return err
			}
		}

		return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
	})
	if err == nil {
		WebhookEventsTotal.WithLabelValues("processed").Inc()
	}
	return err
}

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	h := sha256.New()
	h.Write(payload)
	payloadHash := h.Sum(nil)

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		txQueries := db.New(tx)

		_, err := txQueries.GetWebhookEvent(ctx, db.GetWebhookEventParams{
			Provider:        "crypto",
			ProviderEventID: eventID,
		})
		if err == nil {
			slog.Info("crypto webhook event already processed", "event_id", eventID)
			WebhookEventsTotal.WithLabelValues("duplicate").Inc()
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		_, err = txQueries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
			Provider:        "crypto",
			ProviderEventID: eventID,
			EventType:       eventType,
			PayloadHash:     payloadHash,
			PayloadRedacted: payload,
			Status:          db.PaymentWebhookEventStatusRECEIVED,
			ErrorMessage:    pgtype.Text{},
		})
		if err != nil {
			if coldpath.IsUniqueViolation(err) {
				slog.Info("crypto webhook event deduplicated by unique constraint", "event_id", eventID)
				WebhookEventsTotal.WithLabelValues("duplicate").Inc()
				return nil
			}
			return err
		}

		var intent db.PaymentPaymentIntent
		err = tx.QueryRow(ctx, `
			SELECT id, customer_id, amount_micro, currency, status, provider, provider_ref, idempotency_key, metadata, created_at, updated_at
			FROM payment.payment_intents
			WHERE provider = 'crypto' AND provider_ref = $1
			FOR UPDATE`, providerRef).Scan(
			&intent.ID, &intent.CustomerID, &intent.AmountMicro, &intent.Currency, &intent.Status, &intent.Provider, &intent.ProviderRef, &intent.IdempotencyKey, &intent.Metadata, &intent.CreatedAt, &intent.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Warn("received crypto event for unknown provider_ref", "provider_ref", providerRef)
				return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "unknown provider_ref")
			}
			return err
		}

		if amountMicro <= 0 {
			return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "zero or negative amount")
		}

		if amountMicro < intent.AmountMicro {
			slog.Warn("crypto webhook amount underpay", "intent_id", uuid.UUID(intent.ID.Bytes), "intent_amount", intent.AmountMicro, "webhook_amount", amountMicro)
			if _, err := txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
				ID:          intent.ID,
				Status:      db.PaymentPaymentIntentStatusFAILED,
				ProviderRef: pgtype.Text{String: providerRef, Valid: true},
			}); err != nil {
				return fmt.Errorf("mark underpay intent failed: %w", err)
			}
			return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "underpay")
		}

		if amountMicro < s.cfg.CryptoMinPaymentMicro {
			slog.Warn("crypto webhook amount below minimum", "intent_id", uuid.UUID(intent.ID.Bytes), "min_amount", s.cfg.CryptoMinPaymentMicro, "webhook_amount", amountMicro)
			if _, err := txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
				ID:          intent.ID,
				Status:      db.PaymentPaymentIntentStatusFAILED,
				ProviderRef: pgtype.Text{String: providerRef, Valid: true},
			}); err != nil {
				return fmt.Errorf("mark below-minimum intent failed: %w", err)
			}
			return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "below minimum payment limit")
		}

		if confirmations < s.cfg.CryptoConfirmationDepth {
			slog.Info("crypto webhook pending confirmations", "intent_id", uuid.UUID(intent.ID.Bytes), "confirmations", confirmations, "required", s.cfg.CryptoConfirmationDepth)
			_, err = txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
				ID:          intent.ID,
				Status:      db.PaymentPaymentIntentStatusPROCESSING,
				ProviderRef: pgtype.Text{String: providerRef, Valid: true},
			})
			if err != nil {
				return err
			}
			return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "pending confirmations")
		}

		targetStatus := db.PaymentPaymentIntentStatusSUCCEEDED
		if !isValidTransition(intent.Status, targetStatus) {
			slog.Warn("invalid state transition skipped", "intent_id", uuid.UUID(intent.ID.Bytes), "from", intent.Status, "to", targetStatus)
			return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED,
				fmt.Sprintf("invalid transition from %s to %s", intent.Status, targetStatus))
		}

		alreadySettled := intent.Status == db.PaymentPaymentIntentStatusSUCCEEDED

		_, err = txQueries.UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
			ID:          intent.ID,
			Status:      targetStatus,
			ProviderRef: pgtype.Text{String: providerRef, Valid: true},
		})
		if err != nil {
			return err
		}

		if !alreadySettled {
			holdID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("failed to generate hold id: %w", err)
			}
			releaseAt := time.Now().UTC().Add(14 * 24 * time.Hour)

			_, err = tx.Exec(ctx, `
				INSERT INTO payment.crypto_holds (id, payment_intent_id, customer_id, amount_micro, currency, tx_hash, status, release_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'HELD', $7)
			`, holdID, intent.ID, intent.CustomerID, amountMicro, intent.Currency, txHash, releaseAt)
			if err != nil {
				return fmt.Errorf("failed to create crypto hold: %w", err)
			}
			slog.Info("created crypto hold", "hold_id", holdID, "intent_id", uuid.UUID(intent.ID.Bytes), "release_at", releaseAt)
		}

		if err := checkout.MaybeActivateLicenseFromIntent(ctx, tx, intent); err != nil {
			return fmt.Errorf("license activation: %w", err)
		}

		return updateCryptoWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
	})
	if err == nil {
		WebhookEventsTotal.WithLabelValues("processed").Inc()
	}
	return err
}

func (s *Service) ProcessStripeRefundWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRefundID, paymentIntentRef string, refundAmountMicro int64, refundStatus string) error {
	h := sha256.New()
	h.Write(payload)
	payloadHash := h.Sum(nil)

	redactedBytes, err := coldpath.RedactStripeWebhookPayload(payload)
	if err != nil {
		return fmt.Errorf("redact stripe webhook payload: %w", err)
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		txQueries := db.New(tx)

		_, err := txQueries.GetWebhookEvent(ctx, db.GetWebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: eventID,
		})
		if err == nil {
			slog.Info("refund webhook event already processed", "event_id", eventID)
			WebhookEventsTotal.WithLabelValues("duplicate").Inc()
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		_, err = txQueries.CreateWebhookEvent(ctx, db.CreateWebhookEventParams{
			Provider:        "stripe",
			ProviderEventID: eventID,
			EventType:       eventType,
			PayloadHash:     payloadHash,
			PayloadRedacted: redactedBytes,
			Status:          db.PaymentWebhookEventStatusRECEIVED,
			ErrorMessage:    pgtype.Text{},
		})
		if err != nil {
			if coldpath.IsUniqueViolation(err) {
				slog.Info("refund webhook event deduplicated by unique constraint", "event_id", eventID)
				WebhookEventsTotal.WithLabelValues("duplicate").Inc()
				return nil
			}
			return err
		}

		if refundStatus == "failed" {
			_, lookupErr := txQueries.GetPaymentRefundByProviderRefundID(ctx, db.GetPaymentRefundByProviderRefundIDParams{
				Provider:         "stripe",
				ProviderRefundID: providerRefundID,
			})
			if lookupErr == nil {
				return txQueries.UpdatePaymentRefundStatus(ctx, db.UpdatePaymentRefundStatusParams{
					Provider:         "stripe",
					ProviderRefundID: providerRefundID,
					Status:           db.PaymentRefundStatusFAILED,
				})
			}
			if !errors.Is(lookupErr, pgx.ErrNoRows) {
				return lookupErr
			}
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "refund failed before settlement")
		}

		if refundStatus != "succeeded" {
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "refund not yet succeeded")
		}

		if refundAmountMicro <= 0 {
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "zero or negative refund amount")
		}

		var intent db.PaymentPaymentIntent
		err = tx.QueryRow(ctx, `
			SELECT id, customer_id, amount_micro, currency, status, provider, provider_ref, idempotency_key, metadata, refunded_amount_micro, created_at, updated_at
			FROM payment.payment_intents
			WHERE provider = 'stripe' AND provider_ref = $1
			FOR UPDATE`, paymentIntentRef).Scan(
			&intent.ID, &intent.CustomerID, &intent.AmountMicro, &intent.Currency, &intent.Status, &intent.Provider, &intent.ProviderRef, &intent.IdempotencyKey, &intent.Metadata, &intent.RefundedAmountMicro, &intent.CreatedAt, &intent.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Warn("received stripe refund for unknown payment_intent", "payment_intent", paymentIntentRef)
				return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "unknown payment_intent")
			}
			return err
		}

		if intent.Status != db.PaymentPaymentIntentStatusSUCCEEDED && intent.Status != db.PaymentPaymentIntentStatusREFUNDED {
			slog.Warn("refund webhook for non-settled intent", "intent_id", uuid.UUID(intent.ID.Bytes), "status", intent.Status)
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED,
				fmt.Sprintf("intent status %s not refundable", intent.Status))
		}

		if intent.RefundedAmountMicro+refundAmountMicro > intent.AmountMicro {
			slog.Warn("refund would exceed intent amount", "intent_id", uuid.UUID(intent.ID.Bytes),
				"refunded", intent.RefundedAmountMicro, "delta", refundAmountMicro, "intent_amount", intent.AmountMicro)
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "refund exceeds intent amount")
		}

		existingRefund, err := txQueries.GetPaymentRefundByProviderRefundID(ctx, db.GetPaymentRefundByProviderRefundIDParams{
			Provider:         "stripe",
			ProviderRefundID: providerRefundID,
		})
		if err == nil {
			if existingRefund.Status == db.PaymentRefundStatusSUCCEEDED {
				return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
			}
			return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusIGNORED, "duplicate refund in non-success state")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		refundID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate refund id: %w", err)
		}
		_, err = txQueries.CreatePaymentRefund(ctx, db.CreatePaymentRefundParams{
			ID:               pgtype.UUID{Bytes: refundID, Valid: true},
			PaymentIntentID:  intent.ID,
			Provider:         "stripe",
			ProviderRefundID: providerRefundID,
			AmountMicro:      refundAmountMicro,
			Status:           db.PaymentRefundStatusSUCCEEDED,
		})
		if err != nil {
			if coldpath.IsUniqueViolation(err) {
				return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
			}
			return err
		}

		_, err = txQueries.ApplyIntentRefundAmount(ctx, db.ApplyIntentRefundAmountParams{
			ID:                  intent.ID,
			RefundedAmountMicro: refundAmountMicro,
		})
		if err != nil {
			return err
		}

		intentUUID := uuid.UUID(intent.ID.Bytes)
		customerUUID := uuid.UUID(intent.CustomerID.Bytes)
		outboxPayload, err := settlement.MarshalReverseBalanceOutbox(intentUUID, customerUUID, refundAmountMicro, providerRefundID)
		if err != nil {
			return fmt.Errorf("marshal reverse balance outbox payload: %w", err)
		}
		_, err = txQueries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: settlement.OutboxEventReverseBalance,
			Payload:   outboxPayload,
		})
		if err != nil {
			return err
		}

		return updateStripeWebhookStatus(ctx, txQueries, eventID, db.PaymentWebhookEventStatusPROCESSED, "")
	})
	if err == nil {
		WebhookEventsTotal.WithLabelValues("processed").Inc()
	}
	return err
}
