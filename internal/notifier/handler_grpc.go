package notifier

import (
	"context"
	"fmt"

	"espx/internal/notifier/pb"
)

func (handler *Handler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	result, err := handler.service.SendNotificationInput(ctx, NotificationInputFromPB(req))
	if err != nil {
		return nil, mapRPCError(err)
	}
	return sendNotificationResultToPB(result), nil
}

func (handler *Handler) SendNotificationBatch(ctx context.Context, req *pb.SendNotificationBatchRequest) (*pb.SendNotificationBatchResponse, error) {
	if req == nil || len(req.Notifications) == 0 {
		return nil, mapRPCError(ErrBatchEmpty)
	}
	out := make([]*pb.SendNotificationResponse, 0, len(req.Notifications))
	for _, item := range req.Notifications {
		result, err := handler.service.SendNotificationInput(ctx, NotificationInputFromPB(item))
		if err != nil {
			return nil, mapRPCError(fmt.Errorf("batch item failed: %w", err))
		}
		out = append(out, sendNotificationResultToPB(result))
	}
	return &pb.SendNotificationBatchResponse{Notifications: out}, nil
}

func (handler *Handler) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	notification, err := handler.service.GetNotification(ctx, req.NotificationId)
	if err != nil {
		return nil, mapRPCError(err)
	}
	return &pb.GetNotificationResponse{Notification: NotificationToPB(notification)}, nil
}
