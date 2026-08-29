package payment

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/payment/checkout"
	"ad-event-processor/internal/payment/webhook"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	checkout *checkout.Service
	webhook  *webhook.Service
}

func NewService(pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{
		checkout: checkout.NewService(pool, cfg),
		webhook:  webhook.NewService(pool, cfg),
	}
}

type CreateIntentResult = checkout.CreateIntentResult

func (s *Service) CreatePaymentIntent(ctx context.Context, customerID uuid.UUID, amountMicro int64, currency, idempotencyKey string, metadata map[string]string) (CreateIntentResult, error) {
	return s.checkout.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, metadata)
}

func (s *Service) GetPaymentIntent(ctx context.Context, intentID uuid.UUID) (domain.PaymentIntent, error) {
	return s.checkout.GetPaymentIntent(ctx, intentID)
}

func (s *Service) ListPaymentIntents(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]domain.PaymentIntent, int64, error) {
	return s.checkout.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (s *Service) ProcessStripeWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, rawEvent string) error {
	return s.webhook.ProcessStripeWebhook(ctx, eventID, eventType, payload, providerRef, amountMicro, rawEvent)
}

func (s *Service) ProcessCryptoWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRef string, amountMicro int64, txHash string, confirmations int) error {
	return s.webhook.ProcessCryptoWebhook(ctx, eventID, eventType, payload, providerRef, amountMicro, txHash, confirmations)
}

func (s *Service) ProcessStripeRefundWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerRefundID, paymentIntentRef string, refundAmountMicro int64, refundStatus string) error {
	return s.webhook.ProcessStripeRefundWebhook(ctx, eventID, eventType, payload, providerRefundID, paymentIntentRef, refundAmountMicro, refundStatus)
}

func (s *Service) ProcessStripeDisputeWebhook(ctx context.Context, eventID, eventType string, payload []byte, providerDisputeID, paymentIntentRef string, disputeAmountMicro int64, stripeDisputeStatus string) error {
	return s.webhook.ProcessStripeDisputeWebhook(ctx, eventID, eventType, payload, providerDisputeID, paymentIntentRef, disputeAmountMicro, stripeDisputeStatus)
}

func (s *Service) ListDisputes(ctx context.Context, customerID *uuid.UUID, limit, offset int32) ([]checkout.DisputeListItem, int64, error) {
	return s.webhook.ListDisputes(ctx, customerID, limit, offset)
}

func (s *Service) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	return s.webhook.ReplayWebhook(ctx, provider, providerEventID)
}
