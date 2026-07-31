package billing

import (
	"context"

	"espx/internal/billing/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (handler *Handler) GenerateInvoice(ctx context.Context, req *pb.GenerateInvoiceRequest) (*pb.Invoice, error) {
	customerID, err := parseCustomerID(req.CustomerId)
	if err != nil {
		return nil, err
	}
	if req.BillingMonth == nil {
		return nil, status.Error(codes.InvalidArgument, ErrInvalidBillingMonth.Error())
	}
	return handler.generateInvoice(ctx, customerID, req.BillingMonth.AsTime())
}

func (handler *Handler) GetInvoice(ctx context.Context, req *pb.GetInvoiceRequest) (*pb.Invoice, error) {
	invoiceID, err := parseInvoiceID(req.InvoiceId)
	if err != nil {
		return nil, err
	}
	return handler.getInvoice(ctx, invoiceID)
}

func (handler *Handler) ListInvoices(ctx context.Context, req *pb.ListInvoicesRequest) (*pb.ListInvoicesResponse, error) {
	customerID, err := parseCustomerID(req.CustomerId)
	if err != nil {
		return nil, err
	}
	invoices, total, err := handler.listInvoices(ctx, customerID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	return &pb.ListInvoicesResponse{Invoices: invoices, Total: total}, nil
}
