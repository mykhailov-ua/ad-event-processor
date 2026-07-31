package notifier

import (
	"context"

	"espx/internal/notifier/pb"
)

func sendTestNotification(ctx context.Context, svc *Service, req *pb.SendNotificationRequest) (SendNotificationResult, error) {
	return svc.SendNotificationInput(ctx, NotificationInputFromPB(req))
}

func getTestNotification(ctx context.Context, svc *Service, notificationID string) (Notification, error) {
	return svc.GetNotification(ctx, notificationID)
}
