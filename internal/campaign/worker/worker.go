package worker

import (
	"context"
	"errors"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/runtime"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const scheduleTickBatchTimeout = 2 * time.Minute

const scheduleTickMaxCampaigns = 200

type scheduleLifecycle interface {
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type Worker struct {
	runtime  *runtime.Runtime
	delivery WorkerEffects
}

func NewWorker(rt *runtime.Runtime, delivery WorkerEffects) *Worker {
	return &Worker{runtime: rt, delivery: delivery}
}

func (w *Worker) ProcessScheduleTick(ctx context.Context) error {
	if w == nil || w.runtime == nil {
		return serviceUnavailable()
	}
	opCtx, cancel := context.WithTimeout(ctx, scheduleTickBatchTimeout)
	defer cancel()

	for range scheduleTickMaxCampaigns {
		done, err := w.processNextScheduledCampaign(opCtx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return nil
}

func (w *Worker) processNextScheduledCampaign(ctx context.Context) (done bool, err error) {
	pool := w.runtime.PoolOrNil()
	if pool == nil {
		return true, serviceUnavailable()
	}

	var campID uuid.UUID
	var desired db.CampaignStatusType

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.ClaimScheduledCampaignForUpdate(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				done = true
				return nil
			}
			return err
		}

		var startAt, endAt *time.Time
		if camp.StartAt.Valid {
			startAt = &camp.StartAt.Time
		}
		if camp.EndAt.Valid {
			endAt = &camp.EndAt.Time
		}
		desired = campaign.ResolveScheduleStatus(time.Now(), startAt, endAt)
		if desired == camp.Status {
			return nil
		}
		campID = uuid.UUID(camp.ID.Bytes)
		return nil
	})
	if err != nil || done || campID == uuid.Nil {
		return done, err
	}

	var lifecycle scheduleLifecycle = w.runtime
	if desired == db.CampaignStatusTypeACTIVE {
		if opErr := lifecycle.ResumeCampaign(ctx, campID, "schedule_auto_resume"); opErr != nil {
			return false, nil
		}
	} else if opErr := lifecycle.PauseCampaign(ctx, campID, "schedule_auto_pause"); opErr != nil {
		return false, nil
	}
	return false, nil
}
