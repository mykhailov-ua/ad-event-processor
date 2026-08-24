package postback

import (
	"context"
	"fmt"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func (w *PostbackWorker) resolveDispatchSlot(ctx context.Context, q *db.Queries, hash string, payload PostbackPayload) (dispatchSlot, error) {
	rows, err := q.InsertPostbackDispatchInFlight(ctx, db.InsertPostbackDispatchInFlightParams{
		IdempotencyHash: hash,
		CampaignID:      pgtype.UUID{Bytes: payload.CampaignID, Valid: true},
		ClickID:         payload.ClickID,
		EventType:       payload.EventType,
	})
	if err != nil {
		return dispatchSlotReady, fmt.Errorf("failed to claim dispatch slot: %w", err)
	}
	if rows > 0 {
		return dispatchSlotReady, nil
	}

	existing, err := q.GetPostbackDispatch(ctx, hash)
	if err != nil {
		return dispatchSlotReady, fmt.Errorf("dispatch claim conflict lookup: %w", err)
	}
	switch existing.Status {
	case postbackDispatchStatusSent:
		return dispatchSlotDuplicate, nil
	case postbackDispatchStatusDelivered:
		return dispatchSlotDelivered, nil
	case postbackDispatchStatusInFlight:
		return dispatchSlotReady, nil
	default:
		return dispatchSlotReady, fmt.Errorf("dispatch slot in unexpected status %q", existing.Status)
	}
}

func (w *PostbackWorker) markDispatchDelivered(ctx context.Context, q *db.Queries, hash string) error {
	err := q.UpdatePostbackDispatchStatus(ctx, db.UpdatePostbackDispatchStatusParams{
		IdempotencyHash: hash,
		Status:          postbackDispatchStatusDelivered,
		ErrorMessage:    pgtype.Text{},
		Status_2:        postbackDispatchStatusInFlight,
	})
	if err != nil {
		return fmt.Errorf("failed to mark dispatch delivered: %w", err)
	}
	return nil
}

func (w *PostbackWorker) finalizeDispatchSent(ctx context.Context, q *db.Queries, hash string) error {
	err := q.UpdatePostbackDispatchStatus(ctx, db.UpdatePostbackDispatchStatusParams{
		IdempotencyHash: hash,
		Status:          postbackDispatchStatusSent,
		ErrorMessage:    pgtype.Text{},
		Status_2:        postbackDispatchStatusDelivered,
	})
	if err != nil {
		return fmt.Errorf("failed to mark dispatch sent: %w", err)
	}
	return nil
}
