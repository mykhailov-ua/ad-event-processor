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
)

type Host interface {
	OptimizeHost
	MABMinImpressions() int64
	QueryCreativeBanditStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]flow.CreativeBanditStat, error)
	EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error
}

func ApplyCreativeRuleTx(
	ctx context.Context,
	tx pgx.Tx,
	host Host,
	rule Rule,
	windowEnd time.Time,
) (uuid.UUID, bool, error) {
	if host == nil {
		return uuid.Nil, false, fmt.Errorf("optimize host unavailable")
	}
	if !RuleSupported(rule) || rule.Scope != ScopeCreative || !rule.HasBrand {
		return uuid.Nil, false, nil
	}

	lookbackEnd := windowEnd.UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(rule.LookbackMinutes) * time.Minute)

	q := db.New(tx)
	creatives, err := q.ListActiveBrandCreatives(ctx, domain.ToUUID(rule.BrandID))
	if err != nil {
		return uuid.Nil, false, err
	}
	if len(creatives) < 2 {
		return uuid.Nil, false, nil
	}

	campaignRows, err := q.ListCampaignIDsByBrand(ctx, domain.ToUUID(rule.BrandID))
	if err != nil {
		return uuid.Nil, false, err
	}
	campaignIDs := make([]uuid.UUID, 0, len(campaignRows))
	for _, row := range campaignRows {
		campaignIDs = append(campaignIDs, uuid.UUID(row.Bytes))
	}

	chStats, err := host.QueryCreativeBanditStats(ctx, lookbackStart, lookbackEnd)
	if err != nil {
		return uuid.Nil, false, err
	}
	if chStats == nil {
		return uuid.Nil, false, nil
	}

	creativeIDs := make([]uuid.UUID, 0, len(creatives))
	currentWeights := make(map[uuid.UUID]int32, len(creatives))
	for _, cr := range creatives {
		creativeID := uuid.UUID(cr.ID.Bytes)
		creativeIDs = append(creativeIDs, creativeID)
		currentWeights[creativeID] = cr.Weight
	}

	minImps := host.MABMinImpressions()
	entityStats := flow.AttributeCreativeBanditStats(creativeIDs, campaignIDs, chStats, minImps)
	if len(entityStats) < 2 {
		return uuid.Nil, false, nil
	}

	cfg := flow.BanditApplyConfig{
		MinClicks:         int64(rule.MinClicks),
		MinSpendMicro:     rule.MinSpendMicro,
		Scope:             rule.Scope,
		Algorithm:         rule.Algorithm,
		Objective:         flow.BanditObjectiveROI,
		MaxWeightDeltaPct: rule.MaxWeightDeltaPct,
	}
	newWeights := flow.ApplyCreativeProportionalWeights(creativeIDs, currentWeights, entityStats, cfg)
	if len(newWeights) == 0 {
		return uuid.Nil, false, nil
	}

	changed := false
	weightBatch := &pgx.Batch{}
	for _, cr := range creatives {
		creativeID := uuid.UUID(cr.ID.Bytes)
		newWeight, ok := newWeights[creativeID]
		if !ok || newWeight == cr.Weight {
			continue
		}
		weightBatch.Queue(
			`UPDATE brand_creatives SET weight = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			cr.ID,
			newWeight,
		)
		changed = true
	}
	if !changed {
		return uuid.Nil, false, nil
	}
	br := tx.SendBatch(ctx, weightBatch)
	defer func() { _ = br.Close() }()
	for i := range weightBatch.Len() {
		if _, err := br.Exec(); err != nil {
			return uuid.Nil, false, fmt.Errorf("creative weight batch %d: %w", i, err)
		}
	}

	hash := ApplyActionHash(rule.ID, rule.BrandID, lookbackEnd)
	payload, _ := json.Marshal(map[string]any{
		"rule_id":    rule.ID.String(),
		"scope":      rule.Scope,
		"objective":  rule.Objective,
		"brand_id":   rule.BrandID.String(),
		"window_end": lookbackEnd.Format(time.RFC3339),
	})
	tag, err := db.New(tx).InsertTrafficOptimizerFire(ctx, db.InsertTrafficOptimizerFireParams{
		RuleID:     domain.ToUUID(rule.ID),
		ActionHash: hash,
		BrandID:    domain.ToUUID(rule.BrandID),
		Payload:    payload,
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	if tag == 0 {
		return uuid.Nil, false, nil
	}
	if err := host.EmitBrandCreativesOutbox(ctx, q, rule.BrandID); err != nil {
		return uuid.Nil, false, err
	}
	return rule.BrandID, true, nil
}
