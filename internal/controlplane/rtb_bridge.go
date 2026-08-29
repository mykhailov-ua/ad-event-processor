package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type rtbAdminHost struct {
	svc *Service
}

type rtbRuntimeConfig struct {
	cfg *config.Config
}

func (s *Service) RtbAdminService() *rtbadmin.Service {
	if s == nil {
		return nil
	}
	return rtbadmin.NewService(rtbAdminHost{svc: s})
}

func (h rtbAdminHost) Pool() *pgxpool.Pool { return h.svc.GetPool() }

func (h rtbAdminHost) Config() *config.Config {
	if h.svc == nil {
		return nil
	}
	return h.svc.cfg
}

func (h rtbAdminHost) ErrValidation(msg string) error { return errValidation(msg) }

func (h rtbAdminHost) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h rtbAdminHost) AuditCreateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "CREATE_RTB_DEAL", "rtb_deal", nil, map[string]string{"deal_id": dealID}, nil)
}

func (h rtbAdminHost) AuditUpdateRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "UPDATE_RTB_DEAL", "rtb_deal", nil, map[string]any{"id": id, "deal_id": dealID}, nil)
}

func (h rtbAdminHost) AuditDeleteRtbDeal(ctx context.Context, q db.Querier, adminID uuid.UUID, id int64, dealID string) {
	h.svc.AuditLog(ctx, q, adminID, "DELETE_RTB_DEAL", "rtb_deal", nil, map[string]any{"id": id, "deal_id": dealID}, nil)
}

func (h rtbAdminHost) EnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error {
	payload, err := coldpath.MarshalOutbox(outbox.RtbCatalogReloadPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "RELOAD_RTB_CATALOG",
		Payload:   payload,
	})
	return err
}

func (h rtbAdminHost) UpdateSettings(ctx context.Context, patch map[string]string) error {
	return h.svc.UpdateSettings(ctx, patch)
}

func (h rtbAdminHost) GetSettings(ctx context.Context) (map[string]string, error) {
	return h.svc.GetSettings(ctx)
}

func (h rtbAdminHost) SimulateBidShade(ctx context.Context, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error) {
	if h.svc == nil || h.svc.rtbBidShadeSim == nil {
		return domain.RtbBidShadeOutput{}, fmt.Errorf("rtb bid shade simulator not configured")
	}
	return h.svc.rtbBidShadeSim(ctx, h.svc.GetPool(), h.svc.cfg, in)
}

func (h rtbAdminHost) FloorsPool() *pgxpool.Pool { return h.svc.GetPool() }

func (h rtbAdminHost) FloorsConfig() *config.Config {
	if h.svc == nil {
		return nil
	}
	return h.svc.cfg
}

func (h rtbAdminHost) FloorsClickHouse() *database.ClickHouseQuery {
	if h.svc == nil {
		return nil
	}
	return h.svc.clickhouseQuery
}

func (h rtbAdminHost) FloorsRedisShards() []redis.UniversalClient {
	if h.svc == nil {
		return nil
	}
	return h.svc.redisShards
}

func (h rtbAdminHost) FloorsEnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error {
	return h.EnqueueRtbCatalogReload(ctx, q, trigger)
}

func (s *Service) ListRtbDeals(ctx context.Context) ([]rtbadmin.DealDTO, error) {
	return s.RtbAdminService().ListRtbDeals(ctx)
}

func (s *Service) GetRtbDeal(ctx context.Context, id int64) (rtbadmin.DealDTO, error) {
	return s.RtbAdminService().GetRtbDeal(ctx, id)
}

func (s *Service) CreateRtbDeal(ctx context.Context, spec rtbadmin.DealCreateSpec) (rtbadmin.DealDTO, error) {
	return s.RtbAdminService().CreateRtbDeal(ctx, spec)
}

func (s *Service) UpdateRtbDeal(ctx context.Context, id int64, spec rtbadmin.DealUpdateSpec) (rtbadmin.DealDTO, error) {
	return s.RtbAdminService().UpdateRtbDeal(ctx, id, spec)
}

func (s *Service) DeleteRtbDeal(ctx context.Context, id int64) error {
	return s.RtbAdminService().DeleteRtbDeal(ctx, id)
}

func (s *Service) SetRtbMode(ctx context.Context, mode string) error {
	return s.RtbAdminService().SetRtbMode(ctx, mode)
}

func (s *Service) GetRtbMode(ctx context.Context) string {
	return s.RtbAdminService().GetRtbMode(ctx)
}

func (s *Service) SimulateRtbBidShade(ctx context.Context, req rtbadmin.BidShadeRequest) (rtbadmin.BidShadeResponse, error) {
	out, err := s.RtbAdminService().SimulateRtbBidShade(ctx, rtbadmin.BidShadeRequest{
		GeoHash: req.GeoHash, DeviceType: req.DeviceType, CategoryMask: req.CategoryMask, MinBidMicro: req.MinBidMicro,
	})
	if err != nil {
		return rtbadmin.BidShadeResponse{}, err
	}
	return rtbadmin.BidShadeResponse{
		HasBid: out.HasBid, CampaignID: out.CampaignID, ClearingPriceMicro: out.ClearingPriceMicro,
		RecommendedBidMicro: out.RecommendedBidMicro, ShadeDeltaMicro: out.ShadeDeltaMicro,
		NoBidReason: out.NoBidReason, SecondPriceDeltaPct: out.SecondPriceDeltaPct,
	}, nil
}

func (s *Service) RunFloorOptimizer(ctx context.Context) (int, error) {
	return rtbadmin.RunFloorOptimizer(ctx, rtbAdminHost{svc: s})
}

func (s *Service) ApplyRtbFloorSuggestions(ctx context.Context, dryRun bool, placementIDs []string) (rtbadmin.FloorsApplyResult, error) {
	return rtbadmin.ApplyRtbFloorSuggestions(ctx, rtbAdminHost{svc: s}, dryRun, placementIDs)
}

func (s *Service) OptimizeBidFloors(ctx context.Context) ([]rtbadmin.BidFloorRecommendationDTO, error) {
	return rtbadmin.OptimizeBidFloors(ctx, rtbAdminHost{svc: s})
}

func (s *Service) StartFloorOptimizerWorker(interval time.Duration) {
	if s == nil {
		return
	}
	w := rtbadmin.NewFloorOptimizerWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(s.ctx)
	})
}

func (r rtbRuntimeConfig) RtbMode() string {
	if r.cfg == nil || r.cfg.RtbMode == "" {
		return "off"
	}
	return r.cfg.RtbMode
}

func (r rtbRuntimeConfig) RtbEnabled() bool {
	return r.cfg != nil && r.cfg.RtbEnabled()
}

func (r rtbRuntimeConfig) RtbExchangeNoBidMode() string {
	if r.cfg == nil {
		return ""
	}
	return r.cfg.RtbExchangeNoBidMode
}
