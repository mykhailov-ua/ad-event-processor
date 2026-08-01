package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/billing"
	"espx/internal/config"
)

var _ billing.BillingAPI = (*BillingClient)(nil)

type BillingClient struct {
	api   billing.BillingAPI
	token string
}

func NewBillingClientFromAPI(api billing.BillingAPI, token string) *BillingClient {
	if api == nil || token == "" {
		return nil
	}
	return &BillingClient{api: api, token: token}
}

func NewBillingClientInProcess(api billing.BillingAPI, token string) *BillingClient {
	return NewBillingClientFromAPI(api, token)
}

func openBillingClient(ctx context.Context, cfg *config.Config, opts ServeOptions) (*BillingClient, func(), error) {
	if opts.Billing != nil {
		return opts.Billing, func() {}, nil
	}
	token := ""
	if cfg != nil {
		token = string(cfg.BillingInternalToken)
	}
	api, closeFn, err := billing.OpenAPIOrDial(ctx, cfg)
	if err != nil || api == nil {
		return nil, closeFn, err
	}
	return NewBillingClientFromAPI(api, token), closeFn, nil
}

func (client *BillingClient) Close() error {
	return nil
}

func (client *BillingClient) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GenerateInvoice(ctx, customerID, billingMonth)
}

func (client *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GetInvoice(ctx, invoiceID)
}

func (client *BillingClient) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (billing.ListInvoicesResult, error) {
	if client == nil || client.api == nil {
		return billing.ListInvoicesResult{}, fmt.Errorf("billing client not configured")
	}
	return client.api.ListInvoices(ctx, customerID, limit, offset)
}
