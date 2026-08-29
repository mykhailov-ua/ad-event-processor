package payment

import checkout "ad-event-processor/internal/payment/checkout"

var (
	ErrProviderNotConfigured = checkout.ErrProviderNotConfigured
	ErrIdempotencyConflict   = checkout.ErrIdempotencyConflict
	ErrCheckoutUnavailable   = checkout.ErrCheckoutUnavailable
	ErrCustomerNotFound      = checkout.ErrCustomerNotFound
	ErrInvalidAmount         = checkout.ErrInvalidAmount
	ErrInvalidCustomerID     = checkout.ErrInvalidCustomerID
	ErrInvalidRequestBody    = checkout.ErrInvalidRequestBody
	ErrInvalidIntentID       = checkout.ErrInvalidIntentID
	ErrPaymentIntentNotFound = checkout.ErrPaymentIntentNotFound
	ErrWebhookEventNotFound  = checkout.ErrWebhookEventNotFound
)
