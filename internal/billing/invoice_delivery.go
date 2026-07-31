package billing

import (
	"context"
	"fmt"
	"log/slog"

	"espx/internal/billing/pb"
	"espx/internal/notifier"
	"espx/pkg/branding"
)

type InvoiceDeliverer interface {
	DeliverInvoice(ctx context.Context, customerID, invoiceID, month, currency string, totalMicro int64, pdfURL string) error
}

type NotifierInvoiceDeliverer struct {
	api       notifier.NotifierAPI
	provider  string
	recipient string
	baseURL   string
}

func NewNotifierInvoiceDeliverer(
	api notifier.NotifierAPI,
	provider, recipient, baseURL string,
) *NotifierInvoiceDeliverer {
	if api == nil || recipient == "" {
		return nil
	}
	return &NotifierInvoiceDeliverer{
		api:       api,
		provider:  provider,
		recipient: recipient,
		baseURL:   baseURL,
	}
}

func (d *NotifierInvoiceDeliverer) DeliverInvoice(
	ctx context.Context,
	customerID, invoiceID, month, currency string,
	totalMicro int64,
	pdfURL string,
) error {
	if d == nil || d.api == nil {
		return fmt.Errorf("notifier deliverer not configured")
	}

	title := fmt.Sprintf("Invoice %s", month)
	_, err := d.api.SendNotificationInput(ctx, notifier.NotificationInput{
		Provider:  d.provider,
		Recipient: d.recipient,
		Title:     title,
		TemplateID: "invoice_monthly",
		TemplateVars: map[string]string{
			"customer_id":   customerID,
			"invoice_id":    invoiceID,
			"billing_month": month,
			"currency":      currency,
			"total_micro":   fmt.Sprintf("%d", totalMicro),
		},
		AttachmentURL: pdfURL,
		DedupKey:      fmt.Sprintf("invoice:%s", invoiceID),
	})
	return err
}

type DriftAlerter interface {
	AlertLedgerDrift(ctx context.Context, customerID string, err error)
}

type NotifierDriftAlerter struct {
	deliverer *NotifierInvoiceDeliverer
}

func NewNotifierDriftAlerter(api notifier.NotifierAPI, provider, recipient string) *NotifierDriftAlerter {
	if api == nil || recipient == "" {
		return nil
	}
	return &NotifierDriftAlerter{
		deliverer: &NotifierInvoiceDeliverer{api: api, provider: provider, recipient: recipient},
	}
}

func (a *NotifierDriftAlerter) AlertLedgerDrift(ctx context.Context, customerID string, driftErr error) {
	if a == nil || a.deliverer == nil || driftErr == nil {
		return
	}
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	_, err := a.deliverer.api.SendNotificationInput(ctx, notifier.NotificationInput{
		Provider:  a.deliverer.provider,
		Recipient: a.deliverer.recipient,
		Title:     title,
		Body:      body,
		DedupKey:  fmt.Sprintf("billing:drift:%s", customerID),
	})
	if err != nil {
		slog.Warn("ledger drift alert enqueue failed", "customer_id", customerID, "error", err)
	}
}

func (s *Service) DeliverInvoice(ctx context.Context, inv *pb.Invoice) error {
	if s == nil || inv == nil || s.deliverer == nil {
		return nil
	}
	month := ""
	if inv.BillingMonth != nil {
		month = inv.BillingMonth.AsTime().UTC().Format("2006-01")
	}
	pdfURL := s.invoicePDFURL(inv.Id)
	return s.deliverer.DeliverInvoice(ctx, inv.CustomerId, inv.Id, month, inv.Currency, inv.TotalMicro, pdfURL)
}

func (s *Service) invoicePDFURL(invoiceID string) string {
	if s == nil || s.invoiceBaseURL == "" {
		return ""
	}
	return s.invoiceBaseURL + "/api/v1/billing/invoices/" + invoiceID + "/pdf"
}
