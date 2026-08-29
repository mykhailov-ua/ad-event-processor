package shardadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

func (w *OperationLeaseWorker) RunJanitor(ctx context.Context) (int32, error) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return 0, nil
	}
	ok, err := w.tryJanitorLock(ctx)
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if !ok {
		return 0, nil
	}
	defer w.releaseJanitorLock(ctx)

	var expired int32
	err = w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
		n, err := db.New(w.host.Pool()).OperationLeaseExpireStale(runCtx, defaultOpLeaseExpireBatch)
		if err != nil {
			return err
		}
		expired = n
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if expired > 0 {
		metrics.OpLeaseExpiredTotal.Add(float64(expired))
	}
	return expired, nil
}

func (w *OperationLeaseWorker) tryJanitorLock(ctx context.Context) (bool, error) {
	var ok bool
	err := w.host.Pool().QueryRow(ctx,
		`SELECT pg_try_advisory_lock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	).Scan(&ok)
	return ok, err
}

func (w *OperationLeaseWorker) releaseJanitorLock(ctx context.Context) {
	_, _ = w.host.Pool().Exec(ctx,
		`SELECT pg_advisory_unlock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	)
}

func (w *OperationLeaseWorker) ProcessBooked(ctx context.Context) error {
	if w == nil || w.host == nil || w.executor == nil {
		return nil
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return nil
	}
	q := db.New(w.host.Pool())
	rows, err := q.ListBookedOperationLeasesForNode(ctx, db.ListBookedOperationLeasesForNodeParams{
		NodeID:   w.nodeID,
		RowLimit: 32,
	})
	if err != nil {
		return fmt.Errorf("operation lease process booked node=%s: %w", w.nodeID, err)
	}
	metrics.OpBookedQueueDepth.Set(float64(len(rows)))
	for _, row := range rows {
		opID := uuid.UUID(row.OpID.Bytes)
		if err := w.ExecuteOp(ctx, opID, w.executor); err != nil {
			if errors.Is(err, ErrStaleFencingEpoch) {
				slog.Warn("operation lease stale fencing", "op_id", opID)
				continue
			}
			slog.Warn("operation lease execute failed", "op_id", opID, "err", err)
		}
	}
	return nil
}

func (w *OperationLeaseWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return
	}
	slog.Info("operation lease worker starting",
		"node_id", w.nodeID,
		"role", w.role,
		"region", w.region,
		"timeout_sec", w.timeoutSec,
		"janitor_period", w.janitorPeriod,
	)
	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()
	janitorTicker := time.NewTicker(w.janitorPeriod)
	defer janitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			if err := w.ProcessBooked(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease poll failed", "node_id", w.nodeID, "err", err)
			}
		case <-janitorTicker.C:
			if _, err := w.RunJanitor(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease janitor failed", "node_id", w.nodeID, "err", err)
			}
		}
	}
}
