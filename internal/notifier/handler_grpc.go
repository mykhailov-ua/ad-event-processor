package notifier

import (
	"context"

	"espx/internal/notifier/pb"
)

func (handler *Handler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	resp, err := handler.service.SendNotification(ctx, req)
	return resp, mapRPCError(err)
}

func (handler *Handler) SendNotificationBatch(ctx context.Context, req *pb.SendNotificationBatchRequest) (*pb.SendNotificationBatchResponse, error) {
	resp, err := handler.service.SendNotificationBatch(ctx, req)
	return resp, mapRPCError(err)
}

func (handler *Handler) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	resp, err := handler.service.GetNotification(ctx, req)
	return resp, mapRPCError(err)
}
