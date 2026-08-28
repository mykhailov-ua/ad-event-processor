package ledger

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
)

var _ domain.BillingAPI = (*BillingClient)(nil)

type BillingClient struct {
	api   domain.BillingAPI
	token string
}

func NewBillingClientFromAPI(api domain.BillingAPI, token string) *BillingClient {
	if api == nil || token == "" {
		return nil
	}
	return &BillingClient{api: api, token: token}
}

func NewBillingClientInProcess(api domain.BillingAPI, token string) *BillingClient {
	return NewBillingClientFromAPI(api, token)
}

func (c *BillingClient) Close() error {
	return nil
}

func (c *BillingClient) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*domain.Invoice, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return c.api.GenerateInvoice(ctx, customerID, billingMonth)
}

func (c *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return c.api.GetInvoice(ctx, invoiceID)
}

func (c *BillingClient) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (domain.ListInvoicesResult, error) {
	if c == nil || c.api == nil {
		return domain.ListInvoicesResult{}, fmt.Errorf("billing client not configured")
	}
	return c.api.ListInvoices(ctx, customerID, limit, offset)
}
