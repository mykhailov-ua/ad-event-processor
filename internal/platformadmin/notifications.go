package platformadmin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationsHost interface {
	Pool() *pgxpool.Pool
}

func RetryNotification(ctx context.Context, host NotificationsHost, notificationID string) error {
	if host == nil || host.Pool() == nil {
		return fmt.Errorf("service unavailable")
	}
	id, err := uuid.Parse(notificationID)
	if err != nil {
		return fmt.Errorf("invalid notification id: %w", err)
	}
	tag, err := host.Pool().Exec(ctx, `
		UPDATE notify.notifications
		SET status = 'PENDING',
		 retry_count = 0,
		 error_message = NULL,
		 claimed_at = NULL,
		 updated_at = now()
		WHERE id = $1 AND status = 'FAILED'`, id)
	if err != nil {
		return fmt.Errorf("retry notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.Join(pgx.ErrNoRows, fmt.Errorf("notification not in FAILED state"))
	}
	return nil
}
