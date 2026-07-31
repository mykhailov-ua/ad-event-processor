package payment

import (
	"context"

	"espx/internal/payment/db"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) createPaymentIntent(
	ctx context.Context,
	customerID uuid.UUID,
	amountMicro int64,
	currency, idempotencyKey string,
	metadata map[string]string,
) (CreateIntentResult, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return CreateIntentResult{}, err
	}
	if customerID == uuid.Nil {
		return CreateIntentResult{}, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	if amountMicro <= 0 {
		return CreateIntentResult{}, status.Error(codes.InvalidArgument, "amount_micro must be greater than zero")
	}
	if idempotencyKey == "" {
		return CreateIntentResult{}, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if currency == "" {
		currency = "USD"
	}
	result, err := h.service.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, metadata)
	if err != nil {
		return CreateIntentResult{}, mapPaymentGRPCError(err)
	}
	return result, nil
}

func (h *Handler) getPaymentIntent(ctx context.Context, intentID uuid.UUID) (db.PaymentPaymentIntent, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return db.PaymentPaymentIntent{}, err
	}
	if intentID == uuid.Nil {
		return db.PaymentPaymentIntent{}, status.Error(codes.InvalidArgument, "invalid intent id")
	}
	intent, err := h.service.GetPaymentIntent(ctx, intentID)
	if err != nil {
		return db.PaymentPaymentIntent{}, mapPaymentGRPCError(err)
	}
	return intent, nil
}

func (h *Handler) listPaymentIntents(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]db.PaymentPaymentIntent, int64, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	if customerID == uuid.Nil {
		return nil, 0, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	limit, offset = coldpath.ClampLimitOffset(limit, offset, 10, 100)
	intents, total, err := h.service.ListPaymentIntents(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapPaymentGRPCError(err)
	}
	return intents, total, nil
}

func (h *Handler) listDisputes(ctx context.Context, customerID *uuid.UUID, limit, offset int32) ([]DisputeListItem, int64, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	limit, offset = coldpath.ClampLimitOffset(limit, offset, 10, 100)
	items, total, err := h.service.ListDisputes(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapPaymentGRPCError(err)
	}
	return items, total, nil
}

func (h *Handler) replayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return "", err
	}
	if provider == "" || providerEventID == "" {
		return "", status.Error(codes.InvalidArgument, "provider and provider_event_id are required")
	}
	statusText, err := h.service.ReplayWebhook(ctx, provider, providerEventID)
	if err != nil {
		return "", mapPaymentGRPCError(err)
	}
	return statusText, nil
}
