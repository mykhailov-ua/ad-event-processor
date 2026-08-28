package ledger

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func (h *Handler) generateInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (domain.Invoice, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return domain.Invoice{}, err
	}
	billingMonth = billingMonth.UTC()
	inv, err := h.service.GenerateInvoice(ctx, customerID, billingMonth)
	if err != nil {
		return domain.Invoice{}, mapRPCError(err)
	}
	return inv, nil
}

func (h *Handler) getInvoice(ctx context.Context, invoiceID uuid.UUID) (domain.Invoice, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return domain.Invoice{}, err
	}
	inv, err := h.service.GetInvoice(ctx, invoiceID)
	if err != nil {
		return domain.Invoice{}, mapRPCError(err)
	}
	return inv, nil
}

func (h *Handler) listInvoices(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]domain.Invoice, int64, error) {
	if err := h.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	invoices, total, err := h.service.ListInvoices(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapRPCError(err)
	}
	return invoices, total, nil
}
