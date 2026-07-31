package billing

import (
	"context"
	"time"
)

type InvoiceLine struct {
	LedgerType  string
	AmountMicro int64
	EntryCount  int32
}

type Invoice struct {
	ID            string
	CustomerID    string
	BillingMonth  time.Time
	SubtotalMicro int64
	TaxMicro      int64
	TotalMicro    int64
	Currency      string
	TaxScheme     string
	TaxRateBps    int32
	Lines         []InvoiceLine
	CreatedAt     time.Time
}

type ListInvoicesResult struct {
	Invoices []Invoice
	Total    int64
}

type BillingAPI interface {
	GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*Invoice, error)
	GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
	ListInvoices(ctx context.Context, customerID string, limit, offset int32) (ListInvoicesResult, error)
}
