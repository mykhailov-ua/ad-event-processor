package controlplane

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/payment"
	paymentpb "espx/internal/payment/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var _ payment.PaymentAPI = (*PaymentClient)(nil)

type PaymentClient struct {
	conn  *grpc.ClientConn
	api   payment.PaymentAPI
	token string
}

func NewPaymentClientFromAPI(api payment.PaymentAPI, token string) *PaymentClient {
	if api == nil || token == "" {
		return nil
	}
	return &PaymentClient{api: api, token: token}
}

func NewPaymentClient(cfg *config.Config) (*PaymentClient, error) {
	if cfg == nil || string(cfg.PaymentInternalToken) == "" {
		return nil, nil
	}

	host := cfg.PaymentServerHost
	if host == "" {
		host = "127.0.0.1"
	}
	target := host + ":" + cfg.PaymentServerPort

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("payment gRPC dial %s: %w", target, err)
	}

	token := string(cfg.PaymentInternalToken)
	return &PaymentClient{
		conn:  conn,
		api:   &grpcPaymentAPI{client: paymentpb.NewPaymentServiceClient(conn), token: token},
		token: token,
	}, nil
}

func (c *PaymentClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *PaymentClient) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*payment.CreatePaymentIntentResult, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("payment client not configured")
	}
	return c.api.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
}

func (c *PaymentClient) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (payment.ListPaymentIntentsResult, error) {
	if c == nil || c.api == nil {
		return payment.ListPaymentIntentsResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListPaymentIntents(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (payment.ListDisputesResult, error) {
	if c == nil || c.api == nil {
		return payment.ListDisputesResult{}, fmt.Errorf("payment client not configured")
	}
	return c.api.ListDisputes(ctx, customerID, limit, offset)
}

func (c *PaymentClient) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	if c == nil || c.api == nil {
		return "", fmt.Errorf("payment client not configured")
	}
	return c.api.ReplayWebhook(ctx, provider, providerEventID)
}

type grpcPaymentAPI struct {
	client paymentpb.PaymentServiceClient
	token  string
}

func (g *grpcPaymentAPI) outgoing(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", g.token)
}

func (g *grpcPaymentAPI) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*payment.CreatePaymentIntentResult, error) {
	resp, err := g.client.CreatePaymentIntent(g.outgoing(ctx), &paymentpb.CreatePaymentIntentRequest{
		CustomerId:     customerID,
		AmountMicro:    amountMicro,
		Currency:       currency,
		IdempotencyKey: idempotencyKey,
		Metadata:       meta,
	})
	if err != nil {
		return nil, err
	}
	return payment.CreatePaymentIntentResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (payment.ListPaymentIntentsResult, error) {
	resp, err := g.client.ListPaymentIntents(g.outgoing(ctx), &paymentpb.ListPaymentIntentsRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return payment.ListPaymentIntentsResult{}, err
	}
	return payment.ListPaymentIntentsResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (payment.ListDisputesResult, error) {
	resp, err := g.client.ListDisputes(g.outgoing(ctx), &paymentpb.ListDisputesRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return payment.ListDisputesResult{}, err
	}
	return payment.ListDisputesResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	resp, err := g.client.ReplayWebhook(g.outgoing(ctx), &paymentpb.ReplayWebhookRequest{
		Provider:        provider,
		ProviderEventId: providerEventID,
	})
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	return resp.Status, nil
}
