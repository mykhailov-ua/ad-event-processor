package controlplane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GlobalRegionTrafficScorer struct {
	svc    *Service
	pool   *pgxpool.Pool
	cfg    ScorerConfig
	epoch  atomic.Int64
	states sync.Map
}

func NewGlobalRegionTrafficScorer(svc *Service) *GlobalRegionTrafficScorer {
	cfg := DefaultScorerConfig()
	if svc != nil && svc.cfg != nil {
		cfg = ScorerConfigFrom(svc.cfg)
	}
	return &GlobalRegionTrafficScorer{
		svc:  svc,
		pool: svc.GetPool(),
		cfg:  cfg,
	}
}

func (g *GlobalRegionTrafficScorer) Tick(ctx context.Context, now time.Time) error {
	if g == nil || g.pool == nil {
		return nil
	}
	if g.svc == nil || g.svc.cfg == nil || !g.svc.cfg.MultiRegionGlobal() {
		return nil
	}
	run := func(runCtx context.Context) error {
		return g.tick(runCtx, now, g.epoch.Add(1))
	}
	if g.svc != nil {
		return g.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (g *GlobalRegionTrafficScorer) tick(ctx context.Context, now time.Time, epoch int64) error {
	q := db.New(g.pool)
	regions, err := q.ListActiveRegionCodes(ctx)
	if err != nil {
		return fmt.Errorf("list active region codes: %w", err)
	}
	if len(regions) == 0 {
		return nil
	}

	prevWeights := g.loadPreviousDialWeights(ctx, q)
	inputs := make([]RegionDialInput, 0, len(regions))
	for _, region := range regions {
		nodes, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
			RegionCode: region,
			Role:       RoleTracker,
		})
		if err != nil {
			return fmt.Errorf("list node capacity scores region=%d: %w", region, err)
		}
		if len(nodes) == 0 {
			continue
		}
		var state NodeScoreState
		if v, ok := g.states.Load(region); ok {
			state = v.(NodeScoreState)
		}
		inputs = append(inputs, RegionDialInput{
			RegionCode: region,
			Nodes:      nodes,
			PrevWeight: prevWeights[region],
			State:      state,
		})
	}
	if len(inputs) == 0 {
		return nil
	}

	results := ComputeRegionDialResults(inputs, g.cfg)
	for _, res := range results {
		g.states.Store(res.RegionCode, res.State)
		if err := q.UpsertRegionTrafficDial(ctx, db.UpsertRegionTrafficDialParams{
			RegionCode: res.RegionCode,
			Score:      res.Score,
			Weight:     res.Weight,
			Provenance: res.Provenance,
			EpochID:    epoch,
		}); err != nil {
			return fmt.Errorf("upsert region traffic dial region=%d: %w", res.RegionCode, err)
		}
	}
	return nil
}

func (g *GlobalRegionTrafficScorer) loadPreviousDialWeights(ctx context.Context, q *db.Queries) map[int16]float64 {
	rows, err := q.ListRegionTrafficDial(ctx)
	if err != nil {
		return nil
	}
	out := make(map[int16]float64, len(rows))
	for _, row := range rows {
		out[row.RegionCode] = row.Weight
	}
	return out
}
