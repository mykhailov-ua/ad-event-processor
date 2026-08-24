package payment

import (
	"context"
	"net/http"

	"ad-event-processor/internal/payment/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetPostSettlementMarkHookForTest(hook func(context.Context, db.PaymentPaymentOutbox) error) {
	PostSettlementMarkHook = hook
}

func (outboxWorker *OutboxWorker) ReplacePoolForTest(pool *pgxpool.Pool) {
	outboxWorker.pool = pool
}

func (outboxWorker *OutboxWorker) ReclaimStaleProcessingForTest(ctx context.Context) {
	outboxWorker.reclaimStaleProcessing(ctx)
}

func (webhookHandler *WebhookHandler) HandleCryptoWebhookForTest(w http.ResponseWriter, r *http.Request) {
	webhookHandler.handleCryptoWebhook(w, r)
}
