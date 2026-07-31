package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/billing"
	"espx/internal/billing/pb"
	"espx/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ billing.BillingAPI = (*BillingClient)(nil)

type BillingClient struct {
	conn  *grpc.ClientConn
	api   billing.BillingAPI
	token string
}

func NewBillingClientFromAPI(api billing.BillingAPI, token string) *BillingClient {
	if api == nil || token == "" {
		return nil
	}
	return &BillingClient{api: api, token: token}
}

func NewBillingClient(cfg *config.Config) (*BillingClient, error) {
	if cfg == nil || string(cfg.BillingInternalToken) == "" {
		return nil, nil
	}

	host := cfg.Billing.ServerHost
	if host == "" {
		host = "127.0.0.1"
	}
	target := host + ":" + cfg.Billing.Port

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("billing gRPC dial %s: %w", target, err)
	}

	token := string(cfg.BillingInternalToken)
	return &BillingClient{
		conn:  conn,
		api:   &grpcBillingAPI{client: pb.NewBillingServiceClient(conn), token: token},
		token: token,
	}, nil
}

func (client *BillingClient) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close()
}

func (client *BillingClient) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GenerateInvoice(ctx, customerID, billingMonth)
}

func (client *BillingClient) GetInvoice(ctx context.Context, invoiceID string) (*billing.Invoice, error) {
	if client == nil || client.api == nil {
		return nil, fmt.Errorf("billing client not configured")
	}
	return client.api.GetInvoice(ctx, invoiceID)
}

func (client *BillingClient) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (billing.ListInvoicesResult, error) {
	if client == nil || client.api == nil {
		return billing.ListInvoicesResult{}, fmt.Errorf("billing client not configured")
	}
	return client.api.ListInvoices(ctx, customerID, limit, offset)
}

type grpcBillingAPI struct {
	client pb.BillingServiceClient
	token  string
}

func (g *grpcBillingAPI) outgoing(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", g.token)
}

func (g *grpcBillingAPI) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*billing.Invoice, error) {
	month := time.Date(billingMonth.Year(), billingMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	inv, err := g.client.GenerateInvoice(g.outgoing(ctx), &pb.GenerateInvoiceRequest{
		CustomerId:   customerID,
		BillingMonth: timestamppb.New(month),
	})
	if err != nil {
		return nil, err
	}
	return billing.InvoiceFromPB(inv), nil
}

func (g *grpcBillingAPI) GetInvoice(ctx context.Context, invoiceID string) (*billing.Invoice, error) {
	inv, err := g.client.GetInvoice(g.outgoing(ctx), &pb.GetInvoiceRequest{InvoiceId: invoiceID})
	if err != nil {
		return nil, err
	}
	return billing.InvoiceFromPB(inv), nil
}

func (g *grpcBillingAPI) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (billing.ListInvoicesResult, error) {
	resp, err := g.client.ListInvoices(g.outgoing(ctx), &pb.ListInvoicesRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return billing.ListInvoicesResult{}, err
	}
	return billing.ListInvoicesResultFromPB(resp), nil
}
