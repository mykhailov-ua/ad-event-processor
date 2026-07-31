package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"espx/internal/billing"
	"espx/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerInvariantWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
	notifier LedgerInvariantAlerter
}

type LedgerInvariantAlerter interface {
	AlertLedgerDrift(ctx context.Context, customerID string, err error)
}

func NewLedgerInvariantWorker(pool *pgxpool.Pool, cfg *config.Config, notifier LedgerInvariantAlerter) *LedgerInvariantWorker {
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
	rows, err := w.pool.Query(ctx, `SELECT id FROM customers`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var mismatches int
	for rows.Next() {
		var customerID uuid.UUID
		if err := rows.Scan(&customerID); err != nil {
			continue
		}
		if err := billing.CheckLedgerBalanceInvariant(ctx, w.pool, customerID); err != nil {
			mismatches++
			slog.Error("ledger invariant mismatch", "customer_id", customerID, "err", err)
			if w.notifier != nil {
				w.notifier.AlertLedgerDrift(ctx, customerID.String(), err)
			}
		}
	}
	if mismatches > 0 {
		return errors.New("ledger invariant mismatches detected")
	}
	return rows.Err()
}
