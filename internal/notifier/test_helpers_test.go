package notifier

import (
	"context"
)

func sendTestNotification(ctx context.Context, svc *Service, input NotificationInput) (SendNotificationResult, error) {
	return svc.SendNotificationInput(ctx, input)
}

func getTestNotification(ctx context.Context, svc *Service, notificationID string) (Notification, error) {
	return svc.GetNotification(ctx, notificationID)
}
