package notifier

import (
	"context"
	"espx/internal/notifier/db"

	"github.com/google/uuid"
)

type contextKey string

const NotificationIDContextKey contextKey = "notification_id"

func NotificationIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(NotificationIDContextKey).(string)
	return id, ok
}

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
