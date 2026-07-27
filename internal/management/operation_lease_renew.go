package management

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ErrLeaseRenewExhausted is returned when renew_count reached OP_LEASE_MAX_RENEWALS.
var ErrLeaseRenewExhausted = errors.New("operation lease renew budget exhausted")

// LeaseRenewHook is invoked after a successful PG lease renew (e.g. OpKeyPool flag).
type LeaseRenewHook func(opID uuid.UUID)

func (w *OperationLeaseWorker) renewInterval() time.Duration {
	sec := w.timeoutSec / 3
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
}

func (w *OperationLeaseWorker) runRenewHeartbeat(ctx context.Context, opID uuid.UUID, done <-chan struct{}) {
	if w == nil || !w.renewHeartbeat {
		return
	}
	ticker := time.NewTicker(w.renewInterval())
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RenewLease(ctx, opID); err != nil {
				if errors.Is(err, ErrLeaseRenewExhausted) || errors.Is(err, context.Canceled) {
					return
				}
				slog.Warn("operation lease heartbeat renew failed", "op_id", opID, "error", err)
			}
		}
	}
}

// SetRenewHeartbeat enables or disables automatic renew during ExecuteOp side effects.
func (w *OperationLeaseWorker) SetRenewHeartbeat(enabled bool) {
	if w == nil {
		return
	}
	w.renewHeartbeat = enabled
}

// SetLeaseRenewHook registers a callback after each successful renew.
func (w *OperationLeaseWorker) SetLeaseRenewHook(hook LeaseRenewHook) {
	if w == nil {
		return
	}
	w.onRenew = hook
}
