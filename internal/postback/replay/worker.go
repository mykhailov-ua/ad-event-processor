package replay

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/postback"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultReplayBatch = 20

var replayBackoff = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

// Worker polls DLQ and re-enqueues failed postbacks.
type Worker struct {
	pool      *pgxpool.Pool
	batchSize int32
}

func NewWorker(pool *pgxpool.Pool) *Worker {
	return &Worker{pool: pool, batchSize: defaultReplayBatch}
}

func (w *Worker) ConfigureBatchSize(size int32) {
	if w == nil || size <= 0 {
		return
	}
	w.batchSize = size
}

func AutoReplayEnabled() bool {
	v := strings.TrimSpace(os.Getenv("POSTBACK_AUTO_REPLAY_ENABLED"))
	if v == "" {
		return true
	}
	return strings.EqualFold(v, "true") || v == "1"
}

func (w *Worker) Tick(ctx context.Context) error {
	if w == nil || w.pool == nil || !AutoReplayEnabled() {
		return nil
	}
	q := db.New(w.pool)
	rows, err := q.ListPostbackDLQForAutoReplay(ctx, w.batchSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.replayOne(ctx, q, row); err != nil {
			slog.Warn("postback dlq auto-replay failed", "dlq_id", row.ID, "error", err)
		}
	}
	return nil
}

func (w *Worker) replayOne(ctx context.Context, q *db.Queries, row db.PostbackDlq) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txQ := db.New(tx)

	ev, err := txQ.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   row.Payload,
	})
	if err != nil {
		return err
	}

	nextCount := row.ReplayCount + 1
	nextRetry := nextRetryAt(nextCount)
	status := "RETRIED"
	lastErr := pgtype.Text{String: "auto-replay", Valid: true}
	if err := txQ.UpdatePostbackDLQReplay(ctx, db.UpdatePostbackDLQReplayParams{
		ID:          row.ID,
		ReplayCount: nextCount,
		NextRetryAt: pgtype.Timestamptz{Time: nextRetry, Valid: true},
		Status:      status,
		LastError:   lastErr,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	postback.RecordAutoReplay()
	slog.Info("postback dlq auto-replay enqueued", "dlq_id", row.ID, "outbox_id", ev.ID, "replay_count", nextCount)
	return nil
}

func nextRetryAt(replayCount int32) time.Time {
	idx := int(replayCount) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(replayBackoff) {
		idx = len(replayBackoff) - 1
	}
	return time.Now().UTC().Add(replayBackoff[idx])
}

func Start(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	if pool == nil {
		return
	}
	w := NewWorker(pool)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Tick(ctx); err != nil {
				slog.Warn("postback replay tick failed", "error", err)
			}
		}
	}
}
