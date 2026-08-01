package ledger

import (
	"context"
	"fmt"
	"log/slog"

	"espx/internal/domain"
	"espx/internal/notify"
	"espx/pkg/branding"
)

func (s *Service) SetNotifier(api notify.NotifierAPI, provider, recipient, baseURL string) {
	if s == nil {
		return
	}
	s.notifier = api
	s.notifyProvider = provider
	s.notifyRecipient = recipient
	s.invoiceBaseURL = baseURL
}

func (s *Service) deliverInvoiceNotification(ctx context.Context, customerID, invoiceID, month, currency string, totalMicro int64, pdfURL string) error {
	if s == nil || s.notifier == nil || s.notifyRecipient == "" {
		return fmt.Errorf("notifier deliverer not configured")
	}
	title := fmt.Sprintf("domain.Invoice %s", month)
	_, err := s.notifier.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:   s.notifyProvider,
		Recipient:  s.notifyRecipient,
		Title:      title,
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

func (s *Service) alertLedgerDrift(ctx context.Context, customerID string, driftErr error) {
	if s == nil || s.notifier == nil || s.notifyRecipient == "" || driftErr == nil {
		return
	}
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	_, err := s.notifier.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:  s.notifyProvider,
		Recipient: s.notifyRecipient,
		Title:     title,
		Body:      body,
		DedupKey:  fmt.Sprintf("billing:drift:%s", customerID),
	})
	if err != nil {
		slog.Warn("ledger drift alert enqueue failed", "customer_id", customerID, "error", err)
	}
}

func (s *Service) DeliverInvoice(ctx context.Context, inv domain.Invoice) error {
	if s == nil || s.notifier == nil {
		return nil
	}
	month := inv.BillingMonth.UTC().Format("2006-01")
	pdfURL := s.invoicePDFURL(inv.ID)
	return s.deliverInvoiceNotification(ctx, inv.CustomerID, inv.ID, month, inv.Currency, inv.TotalMicro, pdfURL)
}

func (s *Service) invoicePDFURL(invoiceID string) string {
	if s == nil || s.invoiceBaseURL == "" {
		return ""
	}
	return s.invoiceBaseURL + "/api/v1/billing/invoices/" + invoiceID + "/pdf"
}
