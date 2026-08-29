package entitlements

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

var ErrEntitlementBufferFull = errors.New("entitlement sync buffer full")

type EntitlementSyncHandler func(ctx context.Context, customerID uuid.UUID) error

type EntitlementSyncBuffer struct {
	capacity int
	ch       chan uuid.UUID
	handler  EntitlementSyncHandler
	wg       sync.WaitGroup
}

func NewEntitlementSyncBuffer(capacity int, handler EntitlementSyncHandler) *EntitlementSyncBuffer {
	if capacity <= 0 {
		capacity = 256
	}
	return &EntitlementSyncBuffer{
		capacity: capacity,
		ch:       make(chan uuid.UUID, capacity),
		handler:  handler,
	}
}

func (b *EntitlementSyncBuffer) Start(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-ctx.Done():
				slog.Info("entitlement sync buffer stopping", "reason", ctx.Err())
				return
			case customerID := <-b.ch:
				if err := b.handler(ctx, customerID); err != nil {
					slog.Error("entitlement sync handler failed",
						"customer_id", customerID,
						"error", err,
					)
				}
			}
		}
	}()
}

func (b *EntitlementSyncBuffer) Enqueue(customerID uuid.UUID) error {
	select {
	case b.ch <- customerID:
		return nil
	default:
		slog.Warn("entitlement sync buffer saturated, rejecting enqueue",
			"customer_id", customerID,
			"capacity", b.capacity,
		)
		return ErrEntitlementBufferFull
	}
}

func (b *EntitlementSyncBuffer) Recover(ctx context.Context, customerIDs []uuid.UUID) {
	for _, id := range customerIDs {
		if err := b.Enqueue(id); err != nil {
			slog.Warn("entitlement recovery enqueue dropped",
				"customer_id", id,
				"error", err,
			)
			if err := b.handler(ctx, id); err != nil {
				slog.Error("entitlement recovery direct apply failed",
					"customer_id", id,
					"error", err,
				)
			}
		}
	}
}

func (b *EntitlementSyncBuffer) PendingLen() int {
	return len(b.ch)
}

func (b *EntitlementSyncBuffer) Stop() {
	b.wg.Wait()
}
