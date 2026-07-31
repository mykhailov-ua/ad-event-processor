package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/billing"
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
