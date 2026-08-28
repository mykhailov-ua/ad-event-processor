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

func (w *OutboxWorker) ReplacePoolForTest(pool *pgxpool.Pool) {
	w.pool = pool
}

func (w *OutboxWorker) ReclaimStaleProcessingForTest(ctx context.Context) {
	w.reclaimStaleProcessing(ctx)
}

func (wh *WebhookHandler) HandleCryptoWebhookForTest(w http.ResponseWriter, r *http.Request) {
	wh.handleCryptoWebhook(w, r)
}
