package payment

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrProviderNotConfigured = errors.New("stripe provider not configured")
	ErrIdempotencyConflict   = errors.New("idempotency key conflict")
	ErrCheckoutUnavailable   = errors.New("checkout unavailable")
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrInvalidAmount         = errors.New("invalid payment amount")
	ErrInvalidCustomerID     = errors.New("invalid customer id")
	ErrInvalidRequestBody    = errors.New("invalid request body")
	ErrInvalidIntentID       = errors.New("invalid intent id")
	ErrPaymentIntentNotFound = errors.New("payment intent not found")
	ErrWebhookEventNotFound  = errors.New("webhook event not found")
)

func mapNotFound(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}
