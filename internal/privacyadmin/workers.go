package privacyadmin

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/metrics"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerHost interface {
	ProcessPrivacyErasureTick(ctx context.Context) error
	CleanupConsentEvents(ctx context.Context) error
}

type ErasureWorker struct {
	host WorkerHost
}

func NewErasureWorker(host WorkerHost) *ErasureWorker {
	return &ErasureWorker{host: host}
}

func (w *ErasureWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.ProcessPrivacyErasureTick(ctx); err != nil {
				slog.Error("privacy erasure tick failed", "err", err)
			}
		}
	}
}

type ConsentRetentionWorker struct {
	host WorkerHost
}

func NewConsentRetentionWorker(host WorkerHost) *ConsentRetentionWorker {
	return &ConsentRetentionWorker{host: host}
}

func (w *ConsentRetentionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.CleanupConsentEvents(ctx); err != nil {
				slog.Error("consent retention cleanup failed", "err", err)
			}
		}
	}
}

const (
	eventsRetentionBatchSize = 10_000
	eventsRetentionBatchWait = 100 * time.Millisecond
)

type EventsRetentionWorker struct {
	queries db.Querier
	days    int
}

func NewEventsRetentionWorker(pool *pgxpool.Pool, retentionDays int) *EventsRetentionWorker {
	return &EventsRetentionWorker{
		queries: db.New(pool),
		days:    retentionDays,
	}
}

func (w *EventsRetentionWorker) Start(ctx context.Context) {
	if w == nil || w.days <= 0 {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	w.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *EventsRetentionWorker) RunOnce(ctx context.Context) int64 {
	return w.runOnce(ctx)
}

func (w *EventsRetentionWorker) runOnce(ctx context.Context) int64 {
	cutoff := time.Now().UTC().AddDate(0, 0, -w.days)
	var total int64

	for {
		deleted, err := w.queries.DeleteEventsOlderThanBatch(ctx, db.DeleteEventsOlderThanBatchParams{
			Cutoff:     pgtype.Timestamptz{Time: cutoff, Valid: true},
			BatchLimit: eventsRetentionBatchSize,
		})
		if err != nil {
			slog.Error("events retention batch failed", "err", err, "cutoff", cutoff)
			break
		}
		if deleted > 0 {
			metrics.EventsRetentionDeletedTotal.Add(float64(deleted))
			total += deleted
		}
		if deleted < eventsRetentionBatchSize {
			break
		}
		timer := time.NewTimer(eventsRetentionBatchWait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return total
		case <-timer.C:
		}
	}

	if total > 0 {
		slog.Info("events retention completed", "deleted", total, "retention_days", w.days, "cutoff", cutoff)
	}
	return total
}
