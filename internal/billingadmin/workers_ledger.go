package billingadmin

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageDailyFlushWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

func NewUsageDailyFlushWorker(pool *pgxpool.Pool, interval time.Duration) *UsageDailyFlushWorker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &UsageDailyFlushWorker{pool: pool, interval: interval}
}

func (w *UsageDailyFlushWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("usage daily flush worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Flush(ctx, time.Now().UTC()); err != nil {
				slog.Error("usage daily flush failed", "err", err)
			}
		}
	}
}

func (w *UsageDailyFlushWorker) Flush(ctx context.Context, now time.Time) error {
	usageDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	rows, err := w.pool.Query(ctx, `
		SELECT customer_id, meter, value
		FROM billing.usage_meters
		WHERE period = $1`, period)
	if err != nil {
		return err
	}

	type entry struct {
		custID uuid.UUID
		meter  string
		value  int64
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.custID, &e.meter, &e.value); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	const upsertSQL = `
		INSERT INTO billing.usage_daily (customer_id, usage_date, meter, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (customer_id, usage_date, meter) DO UPDATE
		SET value = EXCLUDED.value`
	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(upsertSQL, e.custID, usageDate, e.meter, e.value)
	}
	br := w.pool.SendBatch(ctx, batch)
	var batchErr error
	for range entries {
		if _, err := br.Exec(); err != nil && batchErr == nil {
			batchErr = err
		}
	}
	if closeErr := br.Close(); closeErr != nil && batchErr == nil {
		batchErr = closeErr
	}
	if batchErr != nil {
		return batchErr
	}
	slog.Info("usage daily flush complete", "date", usageDate.Format("2006-01-02"), "rows", len(entries))
	return nil
}

type LedgerInvariantWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
	notifier InvariantAlerter
}

type InvariantAlerter interface {
	AlertLedgerDrift(ctx context.Context, customerID string, err error)
}

func NewLedgerInvariantWorker(pool *pgxpool.Pool, cfg *config.Config, notifier InvariantAlerter) *LedgerInvariantWorker {
	hours := 24
	if cfg != nil && cfg.LedgerInvariantIntervalHours > 0 {
		hours = cfg.LedgerInvariantIntervalHours
	}
	return &LedgerInvariantWorker{
		pool:     pool,
		interval: time.Duration(hours) * time.Hour,
		notifier: notifier,
	}
}

func (w *LedgerInvariantWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("ledger invariant worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scanAll(ctx); err != nil {
				slog.Error("ledger invariant scan failed", "err", err)
			}
		}
	}
}

func (w *LedgerInvariantWorker) scanAll(ctx context.Context) error {
	mismatches, err := ledger.ListLedgerInvariantMismatches(ctx, w.pool)
	if err != nil {
		return err
	}
	for _, customerID := range mismatches {
		slog.Error("ledger invariant mismatch", "customer_id", customerID)
		if w.notifier != nil {
			w.notifier.AlertLedgerDrift(ctx, customerID.String(), ledger.ErrLedgerDrift)
		}
	}
	if len(mismatches) > 0 {
		return errors.New("ledger invariant mismatches detected")
	}
	return nil
}
