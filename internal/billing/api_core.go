package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (handler *Handler) generateInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (Invoice, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return Invoice{}, err
	}
	billingMonth = billingMonth.UTC()
	inv, err := handler.service.GenerateInvoice(ctx, customerID, billingMonth)
	if err != nil {
		return Invoice{}, mapRPCError(err)
	}
	return inv, nil
}

func (handler *Handler) getInvoice(ctx context.Context, invoiceID uuid.UUID) (Invoice, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return Invoice{}, err
	}
	inv, err := handler.service.GetInvoice(ctx, invoiceID)
	if err != nil {
		return Invoice{}, mapRPCError(err)
	}
	return inv, nil
}

func (handler *Handler) listInvoices(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]Invoice, int64, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	invoices, total, err := handler.service.ListInvoices(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapRPCError(err)
	}
	return invoices, total, nil
}
