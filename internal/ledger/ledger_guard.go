package ledger

import (
	"context"
	"log/slog"
	"time"
)

const ledgerMarginWindow = time.Hour

func (w *Worker) runLedgerMarginCycle(ctx context.Context) error {
	if w == nil || w.pool == nil {
		return nil
	}
	start := time.Now()
	policies, err := w.fetchActivePolicies(ctx)
	if err != nil {
		return err
	}
	if err := w.evaluateLedgerMarginBatch(ctx, policies); err != nil {
		slog.Error("ledger margin batch evaluation failed", "error", err)
		return err
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		slog.Warn("ledger margin guard cycle slow", "duration_ms", elapsed.Milliseconds())
	}
	return nil
}
