package checkout

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
)

var _ domain.PaymentAPI = (*APIClient)(nil)

type APIClient struct {
	api   domain.PaymentAPI
	token string
}

func NewAPIClientFromAPI(api domain.PaymentAPI, token string) *APIClient {
	if api == nil || token == "" {
		return nil
	}
	return &APIClient{api: api, token: token}
}

func NewAPIClientInProcess(api domain.PaymentAPI, token string) *APIClient {
	return NewAPIClientFromAPI(api, token)
}

func (c *APIClient) Close() error {
	return nil
}

func (c *APIClient) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*domain.CreatePaymentIntentResult, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("payment client not configured")
	}
	return c.api.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
}

func (c *APIClient) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (domain.ListPaymentIntentsResult, error) {
	if c == nil || c.api == nil {
		return domain.ListPaymentIntentsResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (c *APIClient) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (domain.ListDisputesResult, error) {
	if c == nil || c.api == nil {
		return domain.ListDisputesResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListDisputes(ctx, customerID, limit, offset)
}

func (c *APIClient) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	if c == nil || c.api == nil {
		return "", fmt.Errorf("payment client not configured")
	}
	return c.api.ReplayWebhook(ctx, provider, providerEventID)
}
