package payment

import (
	"context"

	"espx/internal/payment/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type localGRPCClient struct {
	handler pb.PaymentServiceServer
	token   string
}

func NewLocalGRPCClient(handler pb.PaymentServiceServer, token string) pb.PaymentServiceClient {
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

func (c *localGRPCClient) GetPaymentIntent(ctx context.Context, in *pb.GetPaymentIntentRequest, _ ...grpc.CallOption) (*pb.PaymentIntent, error) {
	return c.handler.GetPaymentIntent(c.incoming(ctx), in)
}

func (c *localGRPCClient) CreatePaymentIntent(ctx context.Context, in *pb.CreatePaymentIntentRequest, _ ...grpc.CallOption) (*pb.CreatePaymentIntentResponse, error) {
	return c.handler.CreatePaymentIntent(c.incoming(ctx), in)
}

func (c *localGRPCClient) ListPaymentIntents(ctx context.Context, in *pb.ListPaymentIntentsRequest, _ ...grpc.CallOption) (*pb.ListPaymentIntentsResponse, error) {
	return c.handler.ListPaymentIntents(c.incoming(ctx), in)
}

func (c *localGRPCClient) ListDisputes(ctx context.Context, in *pb.ListDisputesRequest, _ ...grpc.CallOption) (*pb.ListDisputesResponse, error) {
	return c.handler.ListDisputes(c.incoming(ctx), in)
}

func (c *localGRPCClient) ReplayWebhook(ctx context.Context, in *pb.ReplayWebhookRequest, _ ...grpc.CallOption) (*pb.ReplayWebhookResponse, error) {
	return c.handler.ReplayWebhook(c.incoming(ctx), in)
}
