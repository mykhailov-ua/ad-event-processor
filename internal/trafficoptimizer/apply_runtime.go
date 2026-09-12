package trafficoptimizer

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplyRuleResult struct {
	Applied     bool     `json:"applied"`
	CampaignIDs []string `json:"campaign_ids,omitempty"`
}

func applyRuleNow(
	ctx context.Context,
	pool *pgxpool.Pool,
	host Host,
	pub PublishHost,
	rule Rule,
	now time.Time,
) (bool, []uuid.UUID, error) {
	if pool == nil {
		return false, nil, fmt.Errorf("traffic optimizer pool unavailable")
	}
	if host == nil {
		return false, nil, fmt.Errorf("traffic optimizer host unavailable")
	}
	var publishCampaigns []uuid.UUID
	var applied bool
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if rule.Scope == ScopeCreative {
			_, creativeApplied, err := ApplyCreativeRuleTx(ctx, tx, host, rule, now)
			if err != nil {
				return err
			}
			applied = creativeApplied
			return nil
		}
		campaigns, flowApplied, err := ApplyRuleTx(ctx, tx, host, rule, now)
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
		return false, nil, err
	}
	if applied && pub != nil && rule.Scope != ScopeCreative {
		for _, campID := range publishCampaigns {
			pub.PublishCampaignUpdate(ctx, campID.String())
		}
	}
	if applied {
		WeightUpdatesTotal.WithLabelValues(rule.Scope).Inc()
	}
	return applied, publishCampaigns, nil
}

func (s *RulesService) ApplyRule(ctx context.Context, ruleID uuid.UUID) (ApplyRuleResult, error) {
	if s == nil || s.Pool == nil {
		return ApplyRuleResult{}, ErrUnavailable
	}
	if s.Host == nil {
		return ApplyRuleResult{}, ErrUnavailable
	}
	row, err := db.New(s.Pool).GetTrafficOptimizerRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return ApplyRuleResult{}, fmt.Errorf("rule not found")
	}
	rule, err := RuleFromRow(row)
	if err != nil {
		return ApplyRuleResult{}, err
	}
	applied, campaigns, err := applyRuleNow(ctx, s.Pool, s.Host, s.Publish, rule, time.Now().UTC())
	if err != nil {
		return ApplyRuleResult{}, err
	}
	out := ApplyRuleResult{Applied: applied}
	if len(campaigns) > 0 {
		out.CampaignIDs = make([]string, 0, len(campaigns))
		for _, campID := range campaigns {
			out.CampaignIDs = append(out.CampaignIDs, campID.String())
		}
	}
	return out, nil
}
