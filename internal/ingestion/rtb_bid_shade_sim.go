package ingestion

import (
	"context"

	"espx/internal/config"
	db "espx/internal/domain/db"
	"espx/internal/domain"
	"espx/internal/rtb"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunRtbBidShadeSim(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error) {
	out := domain.RtbBidShadeOutput{}
	if pool == nil {
		out.NoBidReason = rtb.NoBidInvalidRequest.String()
		return out, nil
	}
	registry := NewRegistry(db.New(pool))
	if _, err := registry.Sync(ctx); err != nil {
		return out, err
	}
	if cfg == nil {
		cfg = &config.Config{ClickAmount: 1, ImpressionAmount: 1}
	}
	catalog := NewRtbCatalog(rtb.NewBudgetStore(), BudgetAuthorityShadow)
	catalog.Registry().SetTargetingIndexEnabled(cfg.RtbTargetingIndexEnabled())
	SyncRtbCatalog(ctx, registry, catalog, cfg, nil, RtbBudgetSync{}, nil)

	targeting := RtbTargetingInput{
		GeoHash:             in.GeoHash,
		DeviceType:          in.DeviceType,
		CategoryMask:        in.CategoryMask,
		PublisherFloorMicro: in.MinBidMicro,
	}
	if targeting.CategoryMask == 0 {
		targeting.CategoryMask = 1
	}
	bidReq := BidRequestFromEvent(nil, targeting)
	res, reason := catalog.Registry().RunAuctionEval(&bidReq)
	if !reason.OK() {
		out.NoBidReason = reason.String()
		return out, nil
	}
	uid, ok := catalog.UUIDForWinner(res.CampaignID)
	if !ok {
		out.NoBidReason = rtb.NoBidNoCandidates.String()
		return out, nil
	}
	out.HasBid = true
	out.CampaignID = uid.String()
	out.ClearingPriceMicro = res.Price
	out.RecommendedBidMicro = res.Price - res.Price/50
	if out.RecommendedBidMicro < in.MinBidMicro {
		out.RecommendedBidMicro = in.MinBidMicro
	}
	out.ShadeDeltaMicro = res.Price - out.RecommendedBidMicro
	if res.Price > 0 {
		out.SecondPriceDeltaPct = float64(out.ShadeDeltaMicro) * 100 / float64(res.Price)
	}
	return out, nil
}
