package billing

import (
	"context"
	"time"

	"espx/internal/billing/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseCustomerID(raw string) (uuid.UUID, error) {
	customerID, err := uuid.Parse(raw)
	if err != nil || customerID == uuid.Nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, ErrInvalidCustomerID.Error())
	}
	return customerID, nil
}

func parseInvoiceID(raw string) (uuid.UUID, error) {
	invoiceID, err := uuid.Parse(raw)
	if err != nil || invoiceID == uuid.Nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, ErrInvalidInvoiceID.Error())
	}
	return invoiceID, nil
}

func (handler *Handler) generateInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (*pb.Invoice, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return nil, err
	}
	billingMonth = billingMonth.UTC()
	inv, err := handler.service.GenerateInvoice(ctx, customerID, billingMonth)
	return inv, mapRPCError(err)
}

func (handler *Handler) getInvoice(ctx context.Context, invoiceID uuid.UUID) (*pb.Invoice, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return nil, err
	}
	inv, err := handler.service.GetInvoice(ctx, invoiceID)
	return inv, mapRPCError(err)
}

func (handler *Handler) listInvoices(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]*pb.Invoice, int64, error) {
	if err := handler.requireInternalToken(ctx); err != nil {
		return nil, 0, err
	}
	invoices, total, err := handler.service.ListInvoices(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, mapRPCError(err)
	}
	return invoices, total, nil
}
