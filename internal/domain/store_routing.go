package domain

import (
	"context"
	"fmt"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignRoutingRepo struct {
	pool *pgxpool.Pool
}

func NewCampaignRoutingRepo(pool *pgxpool.Pool) *CampaignRoutingRepo {
	return &CampaignRoutingRepo{pool: pool}
}

func (r *CampaignRoutingRepo) UpsertCampaignRouting(
	ctx context.Context,
	campaignID uuid.UUID,
	homeSlot int16,
	primaryA, primaryB, reserve int16,
	routingEpoch int64,
	hEma, cEma float64,
) (db.CampaignRouting, error) {
	if r.pool == nil {
		return db.CampaignRouting{}, fmt.Errorf("campaign routing repo: nil pool")
	}
	return db.New(r.pool).UpsertCampaignRouting(ctx, db.UpsertCampaignRoutingParams{
		CampaignID:    ToUUID(campaignID),
		HomeSlot:      homeSlot,
		PrimaryAShard: primaryA,
		PrimaryBShard: primaryB,
		ReserveShard:  reserve,
		RoutingEpoch:  routingEpoch,
		HEma:          hEma,
		CEma:          cEma,
	})
}

func (r *CampaignRoutingRepo) GetCampaignRouting(ctx context.Context, campaignID uuid.UUID) (db.CampaignRouting, error) {
	if r.pool == nil {
		return db.CampaignRouting{}, fmt.Errorf("campaign routing repo: nil pool")
	}
	return db.New(r.pool).GetCampaignRouting(ctx, ToUUID(campaignID))
}

func (r *CampaignRoutingRepo) BumpGlobalRoutingEpoch(ctx context.Context) (db.BumpGlobalRoutingEpochRow, error) {
	if r.pool == nil {
		return db.BumpGlobalRoutingEpochRow{}, fmt.Errorf("campaign routing repo: nil pool")
	}
	return db.New(r.pool).BumpGlobalRoutingEpoch(ctx)
}

func (r *CampaignRoutingRepo) GetGlobalRoutingEpoch(ctx context.Context) (db.GetGlobalRoutingEpochRow, error) {
	if r.pool == nil {
		return db.GetGlobalRoutingEpochRow{}, fmt.Errorf("campaign routing repo: nil pool")
	}
	return db.New(r.pool).GetGlobalRoutingEpoch(ctx)
}

func HomeSlotForCampaign(id uuid.UUID) int16 {
	return int16(CampaignSlotIndex(id))
}
