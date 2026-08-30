package trafficoptimizer

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PublishHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

type Worker struct {
	pool                    *pgxpool.Pool
	host                    Host
	pub                     PublishHost
	interval                time.Duration
	evalFloorMinutes        int
	maxEvalsPerCustomerTick int
}

func NewWorker(
	pool *pgxpool.Pool,
	host Host,
	pub PublishHost,
	interval time.Duration,
	evalFloorMinutes int,
	maxEvalsPerCustomerTick int,
) *Worker {
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	if interval > 60*time.Minute {
		interval = 60 * time.Minute
	}
	if evalFloorMinutes < 5 {
		evalFloorMinutes = 5
	}
	if evalFloorMinutes > 60 {
		evalFloorMinutes = 60
	}
	if maxEvalsPerCustomerTick <= 0 {
		maxEvalsPerCustomerTick = 50
	}
	return &Worker{
		pool:                    pool,
		host:                    host,
		pub:                     pub,
		interval:                interval,
		evalFloorMinutes:        evalFloorMinutes,
		maxEvalsPerCustomerTick: maxEvalsPerCustomerTick,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if w == nil || w.pool == nil || w.host == nil {
		return
	}
	recordWorkerTick(time.Now().UTC())
	rules, err := db.New(w.pool).ListEnabledTrafficOptimizerRules(ctx)
	if err != nil {
		slog.Error("traffic optimizer: list rules", "error", err)
		return
	}
	if len(rules) == 0 {
		return
	}
	now := time.Now().UTC()
	evalsByCustomer := make(map[uuid.UUID]int)
	for _, row := range rules {
		rule, err := RuleFromRow(row)
		if err != nil {
			slog.Warn("traffic optimizer: skip rule", "rule_id", row.ID, "error", err)
			continue
		}
		if !RuleDueForEval(now, rule.LastEvaluatedAt, rule.HasLastEvaluated, rule.EvalIntervalMinutes) {
			continue
		}
		if w.maxEvalsPerCustomerTick > 0 && evalsByCustomer[rule.CustomerID] >= w.maxEvalsPerCustomerTick {
			continue
		}
		evalsByCustomer[rule.CustomerID]++
		_ = db.New(w.pool).UpdateTrafficOptimizerRuleLastEvaluated(ctx, db.UpdateTrafficOptimizerRuleLastEvaluatedParams{
			ID:              domain.ToUUID(rule.ID),
			LastEvaluatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		if onCooldown, err := w.ruleOnCooldown(ctx, rule, now); err != nil {
			slog.Warn("traffic optimizer: cooldown check", "rule_id", rule.ID, "error", err)
			continue
		} else if onCooldown {
			EvalTotal.WithLabelValues(rule.CustomerID.String(), "cooldown").Inc()
			continue
		}
		if err := w.applyRule(ctx, rule, now); err != nil {
			slog.Warn("traffic optimizer: apply rule", "rule_id", rule.ID, "error", err)
			EvalTotal.WithLabelValues(rule.CustomerID.String(), "error").Inc()
			continue
		}
		EvalTotal.WithLabelValues(rule.CustomerID.String(), "ok").Inc()
	}
}

func (w *Worker) ruleOnCooldown(ctx context.Context, rule Rule, now time.Time) (bool, error) {
	lastFiredRaw, err := db.New(w.pool).GetTrafficOptimizerRuleLastFiredAt(ctx, domain.ToUUID(rule.ID))
	if err != nil {
		return false, err
	}
	lastFired, ok := lastFiredRaw.(time.Time)
	if !ok {
		return false, nil
	}
	if lastFired.IsZero() {
		return false, nil
	}
	return now.Sub(lastFired) < time.Duration(rule.CooldownMinutes)*time.Minute, nil
}

func (w *Worker) applyRule(ctx context.Context, rule Rule, now time.Time) error {
	var publishCampaigns []uuid.UUID
	var applied bool
	err := pgx.BeginFunc(ctx, w.pool, func(tx pgx.Tx) error {
		if rule.Scope == ScopeCreative {
			_, creativeApplied, err := ApplyCreativeRuleTx(ctx, tx, w.host, rule, now)
			if err != nil {
				return err
			}
			applied = creativeApplied
			return nil
		}
		campaigns, flowApplied, err := ApplyRuleTx(ctx, tx, w.host, rule, now)
		if err != nil {
			return err
		}
		if flowApplied {
			publishCampaigns = uniqueUUIDs(campaigns)
			applied = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}
	WeightUpdatesTotal.WithLabelValues(rule.Scope).Inc()
	if rule.Scope == ScopeCreative {
		return nil
	}
	if w.pub != nil {
		for _, campID := range publishCampaigns {
			w.pub.PublishCampaignUpdate(ctx, campID.String())
		}
	}
	return nil
}

func uniqueUUIDs(in []uuid.UUID) []uuid.UUID {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(in))
	out := make([]uuid.UUID, 0, len(in))
	for _, id := range in {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
