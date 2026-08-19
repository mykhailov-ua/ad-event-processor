package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ledger/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (service *Service) invoiceFromDB(ctx context.Context, invoice db.BillingInvoice) (domain.Invoice, error) {
	lineRows, err := service.queries.ListInvoiceLines(ctx, invoice.ID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("list invoice lines: %w", err)
	}
	return invoiceFromDBRow(invoice, lineRows), nil
}

func (service *Service) invoicesFromDB(ctx context.Context, invoiceRows []db.BillingInvoice) ([]domain.Invoice, error) {
	if len(invoiceRows) == 0 {
		return nil, nil
	}
	ids := make([]pgtype.UUID, len(invoiceRows))
	for i := range invoiceRows {
		ids[i] = invoiceRows[i].ID
	}
	lineRows, err := service.queries.ListInvoiceLinesByInvoiceIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list invoice lines: %w", err)
	}
	linesByInvoice := make(map[[16]byte][]db.BillingInvoiceLine)
	for _, line := range lineRows {
		key := line.InvoiceID.Bytes
		linesByInvoice[key] = append(linesByInvoice[key], line)
	}
	invoices := make([]domain.Invoice, 0, len(invoiceRows))
	for i := range invoiceRows {
		invoices = append(invoices, invoiceFromDBRow(invoiceRows[i], linesByInvoice[invoiceRows[i].ID.Bytes]))
	}
	return invoices, nil
}

func invoiceFromDBRow(invoice db.BillingInvoice, lineRows []db.BillingInvoiceLine) domain.Invoice {
	lines := make([]domain.InvoiceLine, 0, len(lineRows))
	for _, line := range lineRows {
		lines = append(lines, domain.InvoiceLine{
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		})
	}

	monthTime := invoice.BillingMonth.Time.UTC()
	return domain.Invoice{
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
	}
}
