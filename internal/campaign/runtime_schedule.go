package campaign

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func updateCampaignSchedule(
	ctx context.Context,
	pool *pgxpool.Pool,
	fx Effects,
	campaignID uuid.UUID,
	startAt, endAt *time.Time,
	daypartHours []int16,
) error {
	if err := validateDaypartHours(daypartHours); err != nil {
		return err
	}
	if err := validateSchedule(startAt, endAt); err != nil {
		return err
	}
	if pool == nil {
		return errServiceUnavailable()
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		return fx.ApplyCampaignSchedulePatch(ctx, q, campaignID, locked, startAt, endAt, daypartHours)
	})
}
