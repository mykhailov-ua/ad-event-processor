package billing

import (
	"context"

	"espx/internal/billing/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type localGRPCClient struct {
	handler pb.BillingServiceServer
	token   string
}

func NewLocalGRPCClient(handler pb.BillingServiceServer, token string) pb.BillingServiceClient {
	if handler == nil {
		return nil
	}
	return &localGRPCClient{handler: handler, token: token}
}

func (c *localGRPCClient) incoming(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-internal-token", c.token))
}

func (c *localGRPCClient) GenerateInvoice(ctx context.Context, in *pb.GenerateInvoiceRequest, _ ...grpc.CallOption) (*pb.Invoice, error) {
	return c.handler.GenerateInvoice(c.incoming(ctx), in)
}

func (c *localGRPCClient) GetInvoice(ctx context.Context, in *pb.GetInvoiceRequest, _ ...grpc.CallOption) (*pb.Invoice, error) {
	return c.handler.GetInvoice(c.incoming(ctx), in)
}

func (c *localGRPCClient) ListInvoices(ctx context.Context, in *pb.ListInvoicesRequest, _ ...grpc.CallOption) (*pb.ListInvoicesResponse, error) {
	return c.handler.ListInvoices(c.incoming(ctx), in)
}
