package billing

import (
	"context"
	"time"

	"espx/internal/billing/pb"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type grpcBillingAPI struct {
	client pb.BillingServiceClient
	token  string
}

func NewGRPCBillingAPI(client pb.BillingServiceClient, token string) BillingAPI {
	if client == nil || token == "" {
		return nil
	}
	return &grpcBillingAPI{client: client, token: token}
}

func (g *grpcBillingAPI) outgoing(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", g.token)
}

func (g *grpcBillingAPI) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*Invoice, error) {
	month := time.Date(billingMonth.Year(), billingMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	inv, err := g.client.GenerateInvoice(g.outgoing(ctx), &pb.GenerateInvoiceRequest{
		CustomerId:   customerID,
		BillingMonth: timestamppb.New(month),
	})
	if err != nil {
		return nil, err
	}
	return InvoiceFromPB(inv), nil
}

func (g *grpcBillingAPI) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	inv, err := g.client.GetInvoice(g.outgoing(ctx), &pb.GetInvoiceRequest{InvoiceId: invoiceID})
	if err != nil {
		return nil, err
	}
	return InvoiceFromPB(inv), nil
}

func (g *grpcBillingAPI) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (ListInvoicesResult, error) {
	resp, err := g.client.ListInvoices(g.outgoing(ctx), &pb.ListInvoicesRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return ListInvoicesResult{}, err
	}
	return ListInvoicesResultFromPB(resp), nil
}
