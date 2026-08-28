package outbox

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (w *Worker) HandleOutboxEvent(opCtx, ctx context.Context, ev db.OutboxEvent) error {
	return w.handleOutboxEvent(opCtx, ctx, ev)
}

func (w *Worker) CampaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	return w.campaignRemainingBudget(ctx, campaignID)
}

func (w *Worker) SetCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	return w.setCampaignBudgetRemaining(ctx, pipe, campaignIDStr, campaignID, payloadLimit)
}
