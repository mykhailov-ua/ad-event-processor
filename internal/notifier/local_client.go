package notifier

import (
	"context"

	"espx/internal/notifier/pb"

	"google.golang.org/grpc"
)

type localGRPCClient struct {
	handler pb.NotifierServiceServer
}

func NewLocalGRPCClient(handler pb.NotifierServiceServer) pb.NotifierServiceClient {
	if handler == nil {
		return nil
	}
	return &localGRPCClient{handler: handler}
}

func (c *localGRPCClient) SendNotification(ctx context.Context, in *pb.SendNotificationRequest, _ ...grpc.CallOption) (*pb.SendNotificationResponse, error) {
	return c.handler.SendNotification(ctx, in)
}

func (c *localGRPCClient) SendNotificationBatch(ctx context.Context, in *pb.SendNotificationBatchRequest, _ ...grpc.CallOption) (*pb.SendNotificationBatchResponse, error) {
	return c.handler.SendNotificationBatch(ctx, in)
}

func (c *localGRPCClient) GetNotification(ctx context.Context, in *pb.GetNotificationRequest, _ ...grpc.CallOption) (*pb.GetNotificationResponse, error) {
	return c.handler.GetNotification(ctx, in)
}
