package billing

import (
	"context"
	"fmt"
	"time"

	"espx/internal/billing/db"
	"espx/internal/billing/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func InvoiceToPB(inv Invoice) *pb.Invoice {
	lines := make([]*pb.InvoiceLine, 0, len(inv.Lines))
	for _, line := range inv.Lines {
		lines = append(lines, &pb.InvoiceLine{
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		})
	}
	out := &pb.Invoice{
		Id:            inv.ID,
		CustomerId:    inv.CustomerID,
		BillingMonth:  timestamppb.New(inv.BillingMonth),
		SubtotalMicro: inv.SubtotalMicro,
		TaxMicro:      inv.TaxMicro,
		TotalMicro:    inv.TotalMicro,
		Currency:      inv.Currency,
		TaxScheme:     inv.TaxScheme,
		TaxRateBps:    inv.TaxRateBps,
		Lines:         lines,
	}
	if !inv.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(inv.CreatedAt)
	}
	return out
}

func InvoicesToPB(invoices []Invoice) []*pb.Invoice {
	if len(invoices) == 0 {
		return nil
	}
	out := make([]*pb.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, InvoiceToPB(inv))
	}
	return out
}
