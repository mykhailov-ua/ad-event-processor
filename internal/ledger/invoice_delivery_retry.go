package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (service *Service) RetryInvoiceDelivery(ctx context.Context, inv *domain.Invoice, idempotencyKey string) error {
	if service == nil || service.pool == nil {
		return fmt.Errorf("ledger service not configured")
	}
	if inv == nil {
		return fmt.Errorf("invoice required")
	}
	_ = idempotencyKey

	dedupKey := fmt.Sprintf("invoice:%s", inv.ID)
	var notifID string
	err := service.pool.QueryRow(ctx, `
		SELECT id::text
		FROM notify.notifications
		WHERE dedup_key = $1 AND status = 'FAILED'
		ORDER BY created_at DESC
		LIMIT 1`, dedupKey).Scan(&notifID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.DeliverInvoice(ctx, *inv)
		}
		return fmt.Errorf("lookup failed invoice delivery: %w", err)
	}

	tag, err := service.pool.Exec(ctx, `
		UPDATE notify.notifications
		SET status = 'PENDING',
		    retry_count = 0,
		    error_message = NULL,
		    claimed_at = NULL,
		    updated_at = now()
		WHERE id = $1::uuid AND status = 'FAILED'`, notifID)
	if err != nil {
		return fmt.Errorf("retry invoice delivery: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invoice delivery not retryable")
	}
	return nil
}
