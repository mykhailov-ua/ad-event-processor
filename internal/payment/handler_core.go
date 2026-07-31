package payment

import (
	"context"

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
) (createPaymentIntentResult, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return createPaymentIntentResult{}, err
	}
	if customerID == uuid.Nil {
		return createPaymentIntentResult{}, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	if amountMicro <= 0 {
		return createPaymentIntentResult{}, status.Error(codes.InvalidArgument, "amount_micro must be greater than zero")
	}
	if idempotencyKey == "" {
		return createPaymentIntentResult{}, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if currency == "" {
		currency = "USD"
	}
	result, err := h.service.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, metadata)
	if err != nil {
		return createPaymentIntentResult{}, mapPaymentGRPCError(err)
	}
	return createPaymentIntentResult{
		Intent:      paymentIntentFromDB(result.Intent),
		CheckoutURL: result.CheckoutURL,
	}, nil
}

func (h *Handler) getPaymentIntent(ctx context.Context, intentID uuid.UUID) (PaymentIntent, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return PaymentIntent{}, err
	}
	if intentID == uuid.Nil {
		return PaymentIntent{}, status.Error(codes.InvalidArgument, "invalid intent id")
	}
	intent, err := h.service.GetPaymentIntent(ctx, intentID)
	if err != nil {
		return PaymentIntent{}, mapPaymentGRPCError(err)
	}
	return paymentIntentFromDB(intent), nil
}

func (h *Handler) listPaymentIntents(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]PaymentIntent, int64, error) {
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
	if len(intents) == 0 {
		return nil, total, nil
	}
	out := make([]PaymentIntent, 0, len(intents))
	for _, intent := range intents {
		out = append(out, paymentIntentFromDB(intent))
	}
	return out, total, nil
}

func (h *Handler) listDisputes(ctx context.Context, customerID *uuid.UUID, limit, offset int32) ([]Dispute, int64, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	limit, offset = coldpath.ClampLimitOffset(limit, offset, 10, 100)
	items, total, err := h.service.ListDisputes(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapPaymentGRPCError(err)
	}
	if len(items) == 0 {
		return nil, total, nil
	}
	out := make([]Dispute, 0, len(items))
	for _, item := range items {
		out = append(out, disputeFromListItem(item))
	}
	return out, total, nil
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
