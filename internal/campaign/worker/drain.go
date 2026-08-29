package worker

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const drainBatchTimeout = 30 * time.Second

type DrainHost interface {
	WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error
	Pool() *pgxpool.Pool
	DrainWaitTimeoutMs() int64
	FinalizeDrainingCampaign(ctx context.Context, q db.Querier, campaignID uuid.UUID, camp db.Campaign, reason string) error
}

type DrainWorker struct {
	host DrainHost
}

func NewDrainWorker(host DrainHost) *DrainWorker {
	return &DrainWorker{host: host}
}

func (w *DrainWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
				return w.ProcessDraining(runCtx)
			}); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("failed to process draining campaigns", "err", err)
			}
		}
	}
}

func (w *DrainWorker) ProcessDraining(ctx context.Context) error {
	opCtx, cancel := context.WithTimeout(ctx, drainBatchTimeout)
	defer cancel()

	waitTimeoutMs := w.host.DrainWaitTimeoutMs()
	if waitTimeoutMs <= 0 {
		waitTimeoutMs = 100
	}
	threshold := time.Now().Add(-time.Duration(waitTimeoutMs) * time.Millisecond)

	for range 100 {
		finalized, err := w.finalizeNextDraining(opCtx, threshold)
		if err != nil {
			return err
		}
		if !finalized {
			return nil
		}
	}
	return nil
}

func (w *DrainWorker) finalizeNextDraining(ctx context.Context, threshold time.Time) (bool, error) {
	finalized := false
	err := pgx.BeginFunc(ctx, w.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camps, err := q.GetDrainingCampaignsForUpdate(ctx, db.GetDrainingCampaignsForUpdateParams{
			UpdatedAt: pgtype.Timestamptz{Time: threshold, Valid: true},
			Limit:     1,
		})
		if err != nil {
			return err
		}
		if len(camps) == 0 {
			return nil
		}
		camp := camps[0]
		campaignID := uuid.UUID(camp.ID.Bytes)
		if err := w.host.FinalizeDrainingCampaign(ctx, q, campaignID, camp, "Finalized"); err != nil {
			return err
		}
		finalized = true
		return nil
	})
	return finalized, err
}
