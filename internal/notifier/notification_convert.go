package notifier

import (
	"espx/internal/notifier/db"
	"espx/internal/notifier/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func notificationFromDB(n db.NotifierNotification) Notification {
	out := Notification{
		ID:                 uuid.UUID(n.ID.Bytes).String(),
		Provider:           string(n.Provider),
		Recipient:          n.Recipient,
		Body:               n.Body,
		Status:             n.Status,
		RetryCount:         n.RetryCount,
		ErrorMessage:       n.ErrorMessage.String,
		DeliveryMode:       n.DeliveryMode,
		BroadcastProviders: append([]string(nil), n.BroadcastProviders...),
	}
	if n.Title.Valid {
		out.Title = n.Title.String
	}
	if n.DedupKey.Valid {
		out.DedupKey = n.DedupKey.String
	}
	if n.CreatedAt.Valid {
		out.CreatedAt = n.CreatedAt.Time
	}
	if n.UpdatedAt.Valid {
		out.UpdatedAt = n.UpdatedAt.Time
	}
	return out
}

func NotificationToPB(n Notification) *pb.Notification {
	out := &pb.Notification{
		Id:                 n.ID,
		Provider:           MapDBProviderToPB(db.NotifierProvider(n.Provider)),
		Recipient:          n.Recipient,
		Title:              n.Title,
		Body:               n.Body,
		Status:             MapDBStatusToPB(n.Status),
		RetryCount:         n.RetryCount,
		ErrorMessage:       n.ErrorMessage,
		DeliveryMode:       MapDBDeliveryModeToPB(n.DeliveryMode),
		BroadcastProviders: MapDBProvidersToPB(n.BroadcastProviders),
		DedupKey:           n.DedupKey,
	}
	if !n.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(n.CreatedAt)
	}
	if !n.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(n.UpdatedAt)
	}
	return out
}

func sendNotificationResultToPB(result SendNotificationResult) *pb.SendNotificationResponse {
	return &pb.SendNotificationResponse{
		NotificationId: result.NotificationID,
		Status:         MapDBStatusToPB(result.Status),
		Deduplicated:   result.Deduplicated,
	}
}
