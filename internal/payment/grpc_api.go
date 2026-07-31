package payment

import (
	"context"

	"espx/internal/payment/pb"

	"google.golang.org/grpc/metadata"
)

type grpcPaymentAPI struct {
	client pb.PaymentServiceClient
	token  string
}

func NewGRPCPaymentAPI(client pb.PaymentServiceClient, token string) PaymentAPI {
	if client == nil || token == "" {
		return nil
	}
	return &grpcPaymentAPI{client: client, token: token}
}

func (g *grpcPaymentAPI) outgoing(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", g.token)
}

func (g *grpcPaymentAPI) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (*CreatePaymentIntentResult, error) {
	resp, err := g.client.CreatePaymentIntent(g.outgoing(ctx), &pb.CreatePaymentIntentRequest{
		CustomerId:     customerID,
		AmountMicro:    amountMicro,
		Currency:       currency,
		IdempotencyKey: idempotencyKey,
		Metadata:       meta,
	})
	if err != nil {
		return nil, err
	}
	return CreatePaymentIntentResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (ListPaymentIntentsResult, error) {
	resp, err := g.client.ListPaymentIntents(g.outgoing(ctx), &pb.ListPaymentIntentsRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return ListPaymentIntentsResult{}, err
	}
	return ListPaymentIntentsResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ListDisputes(ctx context.Context, customerID string, limit, offset int32) (ListDisputesResult, error) {
	resp, err := g.client.ListDisputes(g.outgoing(ctx), &pb.ListDisputesRequest{
		CustomerId: customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return ListDisputesResult{}, err
	}
	return ListDisputesResultFromPB(resp), nil
}

func (g *grpcPaymentAPI) ReplayWebhook(ctx context.Context, provider, providerEventID string) (string, error) {
	resp, err := g.client.ReplayWebhook(g.outgoing(ctx), &pb.ReplayWebhookRequest{
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
