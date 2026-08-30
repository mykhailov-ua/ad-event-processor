package controlplane

import (
	"context"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/trafficoptimizer"

	"github.com/google/uuid"
)

func (s *Service) TrafficOptimizerRules() *trafficoptimizer.RulesService {
	if s == nil {
		return nil
	}
	floor := 15
	if s.cfg != nil && s.cfg.Management.TrafficOptimizerIntervalMin > 0 {
		floor = s.cfg.Management.TrafficOptimizerIntervalMin
	}
	return &trafficoptimizer.RulesService{
		Pool: s.pool,
		EvalFloorMinutes: func() int {
			return floor
		},
	}
}

func (s *Service) TrafficOptimizerEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Management.TrafficOptimizerEnabled
}

func (s *Service) StartTrafficOptimizerWorker(ctx context.Context, intervalMinutes int) {
	if s == nil || s.cfg == nil || !s.cfg.Management.TrafficOptimizerEnabled {
		return
	}
	if s.ClickHouseQuery() == nil {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	maxEvals := 50
	if s.cfg.Management.TrafficOptimizerMaxEvalsPerCustomerPerTick > 0 {
		maxEvals = s.cfg.Management.TrafficOptimizerMaxEvalsPerCustomerPerTick
	}
	host := trafficOptimizerHost{s}
	w := trafficoptimizer.NewWorker(s.pool, host, s, interval, intervalMinutes, maxEvals)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("traffic optimizer worker enabled", "interval", interval)
}

type trafficOptimizerHost struct {
	svc *Service
}

func (h trafficOptimizerHost) MABLookbackDays() int {
	return h.svc.MABLookbackDays()
}

func (h trafficOptimizerHost) QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	return h.svc.QueryFlowBanditStats(ctx, from, to)
}

func (h trafficOptimizerHost) MABMinImpressions() int64 {
	return h.svc.MABMinImpressions()
}

func (h trafficOptimizerHost) QueryCreativeBanditStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]flow.CreativeBanditStat, error) {
	return h.svc.QueryCreativeBanditStats(ctx, from, to)
}

func (h trafficOptimizerHost) EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	return h.svc.EmitBrandCreativesOutbox(ctx, q, brandID)
}

var (
	_ trafficoptimizer.Host        = trafficOptimizerHost{}
	_ trafficoptimizer.PublishHost = (*Service)(nil)
)
