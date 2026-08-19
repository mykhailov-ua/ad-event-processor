package postback

import (
	"context"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (w *PostbackWorker) loadPostbackConfigs(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]db.PostbackConfig, error) {
	if w == nil || len(campaignIDs) == 0 {
		return nil, nil
	}
	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}
	rows, err := db.New(w.pool).ListPostbackConfigsByCampaignIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]db.PostbackConfig, len(rows))
	for i := range rows {
		id, err := uuid.FromBytes(rows[i].CampaignID.Bytes[:])
		if err != nil {
			continue
		}
		out[id] = rows[i]
	}
	return out, nil
}

func uniqueCampaignIDsFromEvents(events []db.OutboxEvent, payloads []PostbackPayload) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(events))
	out := make([]uuid.UUID, 0, len(events))
	for i := range events {
		if i < len(payloads) {
			id := payloads[i].CampaignID
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
			continue
		}
		payload, err := parsePostbackPayload(events[i].Payload)
		if err != nil {
			continue
		}
		if _, ok := seen[payload.CampaignID]; ok {
			continue
		}
		seen[payload.CampaignID] = struct{}{}
		out = append(out, payload.CampaignID)
	}
	return out
}

func (w *PostbackWorker) batchUpdateOutboxStatus(ctx context.Context, processed, failed, processing []int64) {
	if len(processed) > 0 {
		_, _ = w.pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'PROCESSED', processing_started_at = NULL
			WHERE id = ANY($1)`, processed)
	}
	if len(failed) > 0 {
		_, _ = w.pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'FAILED', processing_started_at = NULL
			WHERE id = ANY($1)`, failed)
	}
	if len(processing) > 0 {
		_, _ = w.pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'PROCESSING', processing_started_at = NOW()
			WHERE id = ANY($1)`, processing)
	}
}
