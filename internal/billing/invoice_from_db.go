package billing

import (
	"context"
	"fmt"
	"time"

	"espx/internal/billing/db"

	"github.com/google/uuid"
)

func (service *Service) invoiceFromDB(ctx context.Context, invoice db.BillingInvoice) (Invoice, error) {
	lineRows, err := service.queries.ListInvoiceLines(ctx, invoice.ID)
	if err != nil {
		return Invoice{}, fmt.Errorf("list invoice lines: %w", err)
	}

	lines := make([]InvoiceLine, 0, len(lineRows))
	for _, line := range lineRows {
		lines = append(lines, InvoiceLine{
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		})
	}

	monthTime := invoice.BillingMonth.Time.UTC()
	return Invoice{
		ID:            uuid.UUID(invoice.ID.Bytes).String(),
		CustomerID:    uuid.UUID(invoice.CustomerID.Bytes).String(),
		BillingMonth:  time.Date(monthTime.Year(), monthTime.Month(), 1, 0, 0, 0, 0, time.UTC),
		SubtotalMicro: invoice.SubtotalMicro,
		TaxMicro:      invoice.TaxMicro,
		TotalMicro:    invoice.TotalMicro,
		Currency:      invoice.Currency,
		TaxScheme:     string(MapSchemeFromDB(invoice.TaxScheme)),
		TaxRateBps:    invoice.TaxRateBps,
		Lines:         lines,
		CreatedAt:     invoice.CreatedAt.Time.UTC(),
	}, nil
}
