package ledger

import (
	"context"
	"fmt"
	"log/slog"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/notify"
	"ad-event-processor/pkg/branding"
)

func (service *Service) SetNotifier(api notify.NotifierAPI, provider, recipient, baseURL string) {
	if service == nil {
		return
	}
	service.notifier = api
	service.notifyProvider = provider
	service.notifyRecipient = recipient
	service.invoiceBaseURL = baseURL
}

func (service *Service) deliverInvoiceNotification(ctx context.Context, customerID, invoiceID, month, currency string, totalMicro int64, pdfURL string) error {
	if service == nil || service.notifier == nil || service.notifyRecipient == "" {
		return fmt.Errorf("notifier deliverer not configured")
	}
	title := fmt.Sprintf("domain.Invoice %s", month)
	_, err := service.notifier.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:   service.notifyProvider,
		Recipient:  service.notifyRecipient,
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

func (service *Service) alertLedgerDrift(ctx context.Context, customerID string, driftErr error) {
	if service == nil || service.notifier == nil || service.notifyRecipient == "" || driftErr == nil {
		return
	}
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	_, err := service.notifier.SendNotificationInput(ctx, notify.NotificationInput{
		Provider:  service.notifyProvider,
		Recipient: service.notifyRecipient,
		Title:     title,
		Body:      body,
		DedupKey:  fmt.Sprintf("billing:drift:%s", customerID),
	})
	if err != nil {
		slog.Warn("ledger drift alert enqueue failed", "customer_id", customerID, "error", err)
	}
}

func (service *Service) DeliverInvoice(ctx context.Context, inv domain.Invoice) error {
	if service == nil || service.notifier == nil {
		return nil
	}
	month := inv.BillingMonth.UTC().Format("2006-01")
	pdfURL := service.invoicePDFURL(inv.ID)
	return service.deliverInvoiceNotification(ctx, inv.CustomerID, inv.ID, month, inv.Currency, inv.TotalMicro, pdfURL)
}

func (service *Service) invoicePDFURL(invoiceID string) string {
	if service == nil || service.invoiceBaseURL == "" {
		return ""
	}
	return service.invoiceBaseURL + "/api/v1/billing/invoices/" + invoiceID + "/pdf"
}
