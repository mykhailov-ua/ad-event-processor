package checkout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment/db"
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

type CreateIntentResult struct {
	Intent         domain.PaymentIntent
	CheckoutURL    string
	DepositAddress string
	DepositNetwork string
	DepositQRSVG   string
}

func (s *Service) CreatePaymentIntent(ctx context.Context, customerID uuid.UUID, amountMicro int64, currency, idempotencyKey string, metadata map[string]string) (CreateIntentResult, error) {
	providerName := DefaultCheckoutProvider(s.cfg)
	if metadata != nil {
		if pName := metadata["provider"]; pName != "" {
			providerName = pName
		}
	}

	intent, claimed, err := s.claimPaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, metadata, providerName)
	if err != nil {
		return CreateIntentResult{}, err
	}
	if !claimed {
		return s.awaitFinalizedIntent(ctx, intent, customerID, amountMicro, currency)
	}

	provRef, checkoutURL, err := CreateCheckout(ctx, s.cfg, providerName, amountMicro, currency, metadata, idempotencyKey)
	if err != nil {
		_ = s.markIntentFailed(ctx, intent.ID)
		if errors.Is(err, ErrProviderNotConfigured) {
			return CreateIntentResult{}, err
		}
		return CreateIntentResult{}, fmt.Errorf("%w: %w", ErrCheckoutUnavailable, err)
	}

	finalized, err := s.finalizePaymentIntent(ctx, intent.ID, provRef, checkoutURL, metadata)
	if err != nil {
		return CreateIntentResult{}, err
	}

	IntentsTotal.WithLabelValues(string(finalized.Status)).Inc()
	return CreateIntentResult{
		Intent:         paymentIntentFromDB(finalized),
		CheckoutURL:    checkoutURL,
		DepositAddress: intentMetadataString(finalized.Metadata, "deposit_address"),
		DepositNetwork: intentMetadataString(finalized.Metadata, "deposit_network"),
		DepositQRSVG:   intentMetadataString(finalized.Metadata, "deposit_qr_svg"),
	}, nil
}

func (s *Service) claimPaymentIntent(
	ctx context.Context,
	customerID uuid.UUID,
	amountMicro int64,
	currency, idempotencyKey string,
	metadata map[string]string,
	providerName string,
) (db.PaymentPaymentIntent, bool, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("acquire conn for idempotency lock: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1::text, 0))`, idempotencyKey); err != nil {
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("idempotency lock: %w", err)
	}
	defer func() {
		unlockCtx := context.WithoutCancel(ctx)
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1::text, 0))`, idempotencyKey)
	}()

	q := db.New(conn)
	existing, err := q.GetPaymentIntentByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("failed to lookup payment intent: %w", err)
	}

	intentID, err := uuid.NewV7()
	if err != nil {
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("generate payment intent id: %w", err)
	}
	metaBytes, err := mergeIntentMetadata(metadata, "")
	if err != nil {
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("failed to encode intent metadata: %w", err)
	}

	var intent db.PaymentPaymentIntent
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		txQueries := db.New(tx)
		var innerErr error
		intent, innerErr = txQueries.CreatePaymentIntent(ctx, db.CreatePaymentIntentParams{
			ID:             pgtype.UUID{Bytes: intentID, Valid: true},
			CustomerID:     pgtype.UUID{Bytes: customerID, Valid: true},
			AmountMicro:    amountMicro,
			Currency:       currency,
			Status:         db.PaymentPaymentIntentStatusCREATED,
			Provider:       providerName,
			ProviderRef:    pgtype.Text{},
			IdempotencyKey: idempotencyKey,
			Metadata:       metaBytes,
		})
		return innerErr
	})
	if err != nil {
		if coldpath.IsUniqueViolation(err) {
			existing, lookupErr := q.GetPaymentIntentByIdempotencyKey(ctx, idempotencyKey)
			if lookupErr != nil {
				return db.PaymentPaymentIntent{}, false, fmt.Errorf("idempotency race recovery failed: %w", lookupErr)
			}
			return existing, false, nil
		}
		return db.PaymentPaymentIntent{}, false, fmt.Errorf("failed to insert payment intent: %w", err)
	}
	return intent, true, nil
}

func (s *Service) finalizePaymentIntent(
	ctx context.Context,
	intentID pgtype.UUID,
	provRef, checkoutURL string,
	metadata map[string]string,
) (db.PaymentPaymentIntent, error) {
	metaBytes, err := mergeIntentMetadata(metadata, checkoutURL)
	if err != nil {
		return db.PaymentPaymentIntent{}, fmt.Errorf("failed to encode intent metadata: %w", err)
	}

	status := db.PaymentPaymentIntentStatusCREATED
	if provRef != "" {
		status = db.PaymentPaymentIntentStatusPENDINGPROVIDER
	}

	var intent db.PaymentPaymentIntent
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE payment.payment_intents
			SET status = $2,
			 provider_ref = COALESCE(NULLIF($3, ''), provider_ref),
			 metadata = $4,
			 updated_at = now()
			WHERE id = $1
			RETURNING id, customer_id, amount_micro, currency, status, provider, provider_ref, idempotency_key, metadata, created_at, updated_at, refunded_amount_micro`,
			intentID, status, provRef, metaBytes)
		return row.Scan(
			&intent.ID, &intent.CustomerID, &intent.AmountMicro, &intent.Currency, &intent.Status,
			&intent.Provider, &intent.ProviderRef, &intent.IdempotencyKey, &intent.Metadata,
			&intent.CreatedAt, &intent.UpdatedAt, &intent.RefundedAmountMicro,
		)
	})
	if err != nil {
		return db.PaymentPaymentIntent{}, fmt.Errorf("failed to finalize payment intent: %w", err)
	}
	return intent, nil
}

func (s *Service) awaitFinalizedIntent(
	ctx context.Context,
	existing db.PaymentPaymentIntent,
	customerID uuid.UUID,
	amountMicro int64,
	currency string,
) (CreateIntentResult, error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if existing.Status != db.PaymentPaymentIntentStatusCREATED || existing.ProviderRef.Valid {
			IntentsTotal.WithLabelValues(string(existing.Status)).Inc()
			return reconcileIdempotentIntent(existing, customerID, amountMicro, currency)
		}
		if time.Now().After(deadline) {
			return CreateIntentResult{}, fmt.Errorf("timeout waiting for payment intent checkout")
		}
		time.Sleep(10 * time.Millisecond)
		refreshed, err := db.New(s.pool).GetPaymentIntentByIdempotencyKey(ctx, existing.IdempotencyKey)
		if err != nil {
			return CreateIntentResult{}, err
		}
		existing = refreshed
	}
}

func (s *Service) markIntentFailed(ctx context.Context, intentID pgtype.UUID) error {
	_, err := db.New(s.pool).UpdatePaymentIntentStatus(ctx, db.UpdatePaymentIntentStatusParams{
		ID:     intentID,
		Status: db.PaymentPaymentIntentStatusFAILED,
	})
	return err
}

func reconcileIdempotentIntent(existing db.PaymentPaymentIntent, customerID uuid.UUID, amountMicro int64, currency string) (CreateIntentResult, error) {
	existCust := uuid.UUID(existing.CustomerID.Bytes)
	if existCust != customerID || existing.AmountMicro != amountMicro || existing.Currency != currency {
		return CreateIntentResult{}, fmt.Errorf("%w: existing intent has customer=%s amount=%d currency=%s", ErrIdempotencyConflict, existCust, existing.AmountMicro, existing.Currency)
	}
	return CreateIntentResult{
		Intent:         paymentIntentFromDB(existing),
		CheckoutURL:    checkoutURLFromIntent(existing),
		DepositAddress: intentMetadataString(existing.Metadata, "deposit_address"),
		DepositNetwork: intentMetadataString(existing.Metadata, "deposit_network"),
		DepositQRSVG:   intentMetadataString(existing.Metadata, "deposit_qr_svg"),
	}, nil
}

func (s *Service) GetPaymentIntent(ctx context.Context, intentID uuid.UUID) (domain.PaymentIntent, error) {
	intent, err := db.New(s.pool).GetPaymentIntent(ctx, pgtype.UUID{Bytes: intentID, Valid: true})
	if err != nil {
		return domain.PaymentIntent{}, mapNotFound(err, ErrPaymentIntentNotFound)
	}
	return paymentIntentFromDB(intent), nil
}

func (s *Service) ListPaymentIntents(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]domain.PaymentIntent, int64, error) {
	q := db.New(s.pool)
	custUUID := pgtype.UUID{Bytes: customerID, Valid: true}
	listParams := db.ListPaymentIntentsParams{
		CustomerID: custUUID,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountPaymentIntents(ctx, custUUID) },
		func() ([]db.PaymentPaymentIntent, error) { return q.ListPaymentIntents(ctx, listParams) },
		paymentIntentFromDB,
	)
}
