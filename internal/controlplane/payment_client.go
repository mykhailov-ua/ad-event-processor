package controlplane

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/payment"
)

var _ payment.PaymentAPI = (*PaymentClient)(nil)

type PaymentClient struct {
	api   payment.PaymentAPI
	token string
}

func NewPaymentClientFromAPI(api payment.PaymentAPI, token string) *PaymentClient {
	if api == nil || token == "" {
		return nil
	}
	return &PaymentClient{api: api, token: token}
}

func NewPaymentClientInProcess(api payment.PaymentAPI, token string) *PaymentClient {
	return NewPaymentClientFromAPI(api, token)
}

func openPaymentClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*PaymentClient, func(), error) {
	if opts.Payment != nil {
		return opts.Payment, func() {}, nil
	}
	token := ""
	if cfg != nil {
		token = string(cfg.PaymentInternalToken)
	}
	api, closeFn, err := payment.OpenAPIOrDial(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewPaymentClientFromAPI(api, token), closeFn, nil
}

func (c *PaymentClient) Close() error {
	return nil
}

func (c *PaymentClient) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*payment.CreatePaymentIntentResult, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("payment client not configured")
	}
	return c.api.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
}

func (c *PaymentClient) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (payment.ListPaymentIntentsResult, error) {
	if c == nil || c.api == nil {
		return payment.ListPaymentIntentsResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (payment.ListDisputesResult, error) {
	if c == nil || c.api == nil {
		return payment.ListDisputesResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListDisputes(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	if c == nil || c.api == nil {
		return "", fmt.Errorf("payment client not configured")
	}
	return c.api.ReplayWebhook(ctx, provider, providerEventID)
}
