package trafficoptimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type OptimizeHost interface {
	flow.BanditHost
}

func ApplyRuleTx(
	ctx context.Context,
	tx pgx.Tx,
	host OptimizeHost,
	rule Rule,
	windowEnd time.Time,
) ([]uuid.UUID, bool, error) {
	if host == nil {
		return nil, false, fmt.Errorf("optimize host unavailable")
	}
	if !RuleSupported(rule) {
		return nil, false, nil
	}
	if rule.Scope == ScopeCreative {
		return nil, false, nil
	}

	lookbackEnd := windowEnd.UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(rule.LookbackMinutes) * time.Minute)

	filter := flow.BanditFlowFilter{}
	if rule.HasFlow {
		filter.FlowID = &rule.FlowID
	}
	if rule.HasCampaign {
		filter.CampaignID = &rule.CampaignID
	}

	cfg := flow.BanditApplyConfig{
		MinClicks:         int64(rule.MinClicks),
		MinSpendMicro:     rule.MinSpendMicro,
		Scope:             rule.Scope,
		Algorithm:         rule.Algorithm,
		Objective:         ruleObjectiveForFlow(rule.Objective),
		MaxWeightDeltaPct: rule.MaxWeightDeltaPct,
	}

	publishCampaigns, err := flow.OptimizeFlowBanditFilteredTx(ctx, tx, host, filter, cfg, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, false, err
	}
	if len(publishCampaigns) == 0 {
		return nil, false, nil
	}

	flowKey := rule.FlowID
	if !rule.HasFlow {
		flowKey = uuid.Nil
	}
	hash := ApplyActionHash(rule.ID, flowKey, lookbackEnd)
	payload, _ := json.Marshal(map[string]any{
		"rule_id":    rule.ID.String(),
		"scope":      rule.Scope,
		"objective":  rule.Objective,
		"campaigns":  publishCampaigns,
		"window_end": lookbackEnd.Format(time.RFC3339),
	})

	var campParam, flowParam, brandParam pgtype.UUID
	if rule.HasCampaign {
		campParam = domain.ToUUID(rule.CampaignID)
	}
	if rule.HasFlow {
		flowParam = domain.ToUUID(rule.FlowID)
	}
	if rule.HasBrand {
		brandParam = domain.ToUUID(rule.BrandID)
	}

	tag, err := db.New(tx).InsertTrafficOptimizerFire(ctx, db.InsertTrafficOptimizerFireParams{
		RuleID:     domain.ToUUID(rule.ID),
		ActionHash: hash,
		CampaignID: campParam,
		FlowID:     flowParam,
		BrandID:    brandParam,
		Payload:    payload,
	})
	if err != nil {
		return nil, false, err
	}
	if tag == 0 {
		return nil, false, nil
	}
	return publishCampaigns, true, nil
}

func ruleObjectiveForFlow(objective string) string {
	switch objective {
	case ObjectiveEPC:
		return flow.BanditObjectiveEPC
	case ObjectiveRevenue:
		return flow.BanditObjectiveRevenue
	case ObjectiveROI:
		return flow.BanditObjectiveROI
	default:
		return objective
	}
}
