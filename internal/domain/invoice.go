package domain

import (
	"context"
	"encoding/json"
	"time"
)

type InvoiceLine struct {
	LedgerType  string `json:"ledger_type"`
	AmountMicro int64  `json:"amount_micro"`
	EntryCount  int32  `json:"entry_count"`
}

type Invoice struct {
	ID            string        `json:"id"`
	CustomerID    string        `json:"customer_id"`
	BillingMonth  time.Time     `json:"-"`
	SubtotalMicro int64         `json:"subtotal_micro"`
	TaxMicro      int64         `json:"tax_micro"`
	TotalMicro    int64         `json:"total_micro"`
	Currency      string        `json:"currency"`
	TaxScheme     string        `json:"tax_scheme"`
	TaxRateBps    int32         `json:"tax_rate_bps"`
	Lines         []InvoiceLine `json:"lines"`
	CreatedAt     time.Time     `json:"-"`
	PDFURL        string        `json:"pdf_url,omitempty"`
}

func (i Invoice) MarshalJSON() ([]byte, error) {
	month := ""
	if !i.BillingMonth.IsZero() {
		month = i.BillingMonth.UTC().Format("2006-01")
	}
	return json.Marshal(struct {
		ID            string        `json:"id"`
		CustomerID    string        `json:"customer_id"`
		BillingMonth  string        `json:"billing_month"`
		SubtotalMicro int64         `json:"subtotal_micro"`
		TaxMicro      int64         `json:"tax_micro"`
		TotalMicro    int64         `json:"total_micro"`
		Currency      string        `json:"currency"`
		TaxScheme     string        `json:"tax_scheme"`
		TaxRateBps    int32         `json:"tax_rate_bps"`
		Lines         []InvoiceLine `json:"lines"`
		PDFURL        string        `json:"pdf_url,omitempty"`
	}{
		ID:            i.ID,
		CustomerID:    i.CustomerID,
		BillingMonth:  month,
		SubtotalMicro: i.SubtotalMicro,
		TaxMicro:      i.TaxMicro,
		TotalMicro:    i.TotalMicro,
		Currency:      i.Currency,
		TaxScheme:     i.TaxScheme,
		TaxRateBps:    i.TaxRateBps,
		Lines:         i.Lines,
		PDFURL:        i.PDFURL,
	})
}

type ListInvoicesResult struct {
	Invoices []Invoice `json:"invoices"`
	Total    int64     `json:"total"`
}

type InvoiceSummary struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id,omitempty"`
	BillingMonth  string `json:"billing_month"`
	SubtotalMicro int64  `json:"subtotal_micro"`
	TaxMicro      int64  `json:"tax_micro"`
	TotalMicro    int64  `json:"total_micro"`
	Status        string `json:"status"`
	Currency      string `json:"currency"`
}

type PaymentSummary struct {
	LedgerID        int64  `json:"ledger_id"`
	AmountMicro     int64  `json:"amount_micro"`
	PaymentIntentID string `json:"payment_intent_id,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type BillingAPI interface {
	GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*Invoice, error)
	GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
	ListInvoices(ctx context.Context, customerID string, limit, offset int32) (ListInvoicesResult, error)
}
