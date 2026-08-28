package campaign

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const deliveryTickBatchTimeout = 2 * time.Minute

func (w *Worker) RunAutoscaleBudgetTick(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return RunAutoscaleBudgetsTick(ctx, w.delivery, syncWorkers)
}

func (w *Worker) RunDeliveryOptimizerTick(ctx context.Context, syncWorkers []*domain.SyncWorker, runMAB bool) error {
	if w == nil || w.delivery == nil {
		return errServiceUnavailable()
	}
	return w.delivery.RunWithPostgresLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := context.WithTimeout(runCtx, deliveryTickBatchTimeout)
		defer cancel()

		syncRegistryWorkers(syncWorkers, opCtx)

		merge := make(DeliveryOutboxMerge)
		var mabBrands []uuid.UUID
		var flowBanditCampaigns []uuid.UUID

		err := pgx.BeginFunc(opCtx, w.delivery.Pool(), func(tx pgx.Tx) error {
			if err := closedLoopPacingControllerTx(opCtx, tx, merge, w.delivery); err != nil {
				return err
			}
			if err := AutoscaleBudgetsTx(opCtx, tx, merge, w.delivery); err != nil {
				return err
			}
			if runMAB {
				brands, err := OptimizeBrandCreativeMABTx(opCtx, tx, w.delivery)
				if err != nil {
					return err
				}
				mabBrands = brands
				campaigns, err := flow.OptimizeFlowBanditTx(opCtx, tx, flowBanditAdapter{host: w.delivery})
				if err != nil {
					return err
				}
				flowBanditCampaigns = campaigns
			}
			if err := merge.Flush(opCtx, tx); err != nil {
				return err
			}
			for _, brandID := range mabBrands {
				if err := w.delivery.EmitBrandCreativesOutbox(opCtx, db.New(tx), brandID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, campID := range flowBanditCampaigns {
			w.delivery.PublishCampaignUpdate(opCtx, campID.String())
		}
		return nil
	})
}

func (w *Worker) ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	if w == nil || w.delivery == nil {
		return errServiceUnavailable()
	}
	return w.delivery.RunWithPostgresLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := context.WithTimeout(runCtx, deliveryTickBatchTimeout)
		defer cancel()

		syncRegistryWorkers(syncWorkers, opCtx)

		return pgx.BeginFunc(opCtx, w.delivery.Pool(), func(tx pgx.Tx) error {
			return closedLoopPacingControllerTx(opCtx, tx, nil, w.delivery)
		})
	})
}

func syncRegistryWorkers(syncWorkers []*domain.SyncWorker, opCtx context.Context) {
	for _, sw := range syncWorkers {
		if sw != nil {
			sw.SyncAll(opCtx)
		}
	}
}
