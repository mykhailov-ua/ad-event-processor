package worker

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
)

type LoopHost interface {
	ProcessScheduleTick(ctx context.Context) error
	ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error
	RunVPPPacingController(ctx context.Context) error
	AutoscaleBudgets(ctx context.Context, syncWorkers []*domain.SyncWorker) error
	RunDeliveryOptimizerTick(ctx context.Context, syncWorkers []*domain.SyncWorker, runMAB bool) error
	MABInterval() time.Duration
}
