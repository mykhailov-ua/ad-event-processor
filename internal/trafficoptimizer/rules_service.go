package trafficoptimizer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RulesService struct {
	Pool                  *pgxpool.Pool
	EvalFloorMinutes      func() int
	AllowExtendedLookback func() bool
}

func (s *RulesService) ListRules(ctx context.Context, customerID uuid.UUID) ([]RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return nil, ErrUnavailable
	}
	rows, err := db.New(s.Pool).ListTrafficOptimizerRulesByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]RuleDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ruleToDTO(row))
	}
	return out, nil
}

func (s *RulesService) CreateRule(ctx context.Context, req UpsertRuleRequest) (RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return RuleDTO{}, ErrUnavailable
	}
	req, err := ApplyPreset(req)
	if err != nil {
		return RuleDTO{}, err
	}
	params, err := s.buildRuleParams(req)
	if err != nil {
		return RuleDTO{}, err
	}
	row, err := db.New(s.Pool).InsertTrafficOptimizerRule(ctx, params)
	if err != nil {
		return RuleDTO{}, err
	}
	return ruleToDTO(row), nil
}

func (s *RulesService) UpdateRule(ctx context.Context, ruleID uuid.UUID, req UpsertRuleRequest) (RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return RuleDTO{}, ErrUnavailable
	}
	req, err := ApplyPreset(req)
	if err != nil {
		return RuleDTO{}, err
	}
	params, err := s.buildRuleParams(req)
	if err != nil {
		return RuleDTO{}, err
	}
	row, err := db.New(s.Pool).UpdateTrafficOptimizerRule(ctx, db.UpdateTrafficOptimizerRuleParams{
		ID:                  domain.ToUUID(ruleID),
		CustomerID:          params.CustomerID,
		CampaignID:          params.CampaignID,
		FlowID:              params.FlowID,
		BrandID:             params.BrandID,
		Name:                params.Name,
		Scope:               params.Scope,
		Objective:           params.Objective,
		Algorithm:           params.Algorithm,
		LookbackMinutes:     params.LookbackMinutes,
		MinClicks:           params.MinClicks,
		MinConversions:      params.MinConversions,
		MinSpendMicro:       params.MinSpendMicro,
		EvalIntervalMinutes: params.EvalIntervalMinutes,
		CooldownMinutes:     params.CooldownMinutes,
		MaxWeightDeltaPct:   params.MaxWeightDeltaPct,
		PresetKey:           params.PresetKey,
		PresetParameters:    params.PresetParameters,
		Enabled:             params.Enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuleDTO{}, fmt.Errorf("rule not found")
		}
		return RuleDTO{}, err
	}
	return ruleToDTO(row), nil
}

func (s *RulesService) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	if s == nil || s.Pool == nil {
		return ErrUnavailable
	}
	tag, err := db.New(s.Pool).DeleteTrafficOptimizerRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return err
	}
	if tag == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *RulesService) DryRunRule(ctx context.Context, ruleID uuid.UUID) (DryRunResponse, error) {
	if s == nil || s.Pool == nil {
		return DryRunResponse{}, ErrUnavailable
	}
	row, err := db.New(s.Pool).GetTrafficOptimizerRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return DryRunResponse{}, fmt.Errorf("rule not found")
	}
	if _, err := RuleFromRow(row); err != nil {
		return DryRunResponse{}, err
	}
	return DryRunResponse{StaleWeights: true, Arms: []DryRunArmResult{}}, nil
}

func (s *RulesService) evalFloorMinutes() int {
	if s != nil && s.EvalFloorMinutes != nil {
		floor := s.EvalFloorMinutes()
		if floor < 5 {
			return 5
		}
		if floor > 60 {
			return 60
		}
		return floor
	}
	return 15
}

func (s *RulesService) allowExtendedLookback() bool {
	if s != nil && s.AllowExtendedLookback != nil {
		return s.AllowExtendedLookback()
	}
	return false
}

func (s *RulesService) buildRuleParams(req UpsertRuleRequest) (db.InsertTrafficOptimizerRuleParams, error) {
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, fmt.Errorf("invalid customer_id")
	}
	scope, err := NormalizeScope(req.Scope)
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	objective, err := NormalizeObjective(req.Objective)
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	algorithm, err := NormalizeAlgorithm(req.Algorithm)
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	if err := ValidateObjectiveAlgorithmPair(objective, algorithm); err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return db.InsertTrafficOptimizerRuleParams{}, fmt.Errorf("name is required")
	}
	if err := ValidateRuleTargets(scope, req.BrandID, req.CampaignID, req.FlowID); err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	evalInterval, err := NormalizeEvalIntervalMinutes(req.EvalIntervalMinutes, s.evalFloorMinutes())
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	presetParams, err := encodePresetParameters(req.PresetParameters)
	if err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	lookback := ClampLookbackMinutes(req.LookbackMinutes)
	if req.LookbackMinutes == 0 {
		lookback = 1440
	}
	minClicks := req.MinClicks
	if minClicks == 0 {
		minClicks = 100
	}
	if err := ValidateMinClicks(minClicks); err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	minSpend := req.MinSpendMicro
	if minSpend == 0 && objective == ObjectiveROI {
		minSpend = 1_000_000
	}
	if err := ValidateMinSpendMicro(objective, minSpend); err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	if err := ValidateLookbackMinutes(lookback, s.allowExtendedLookback()); err != nil {
		return db.InsertTrafficOptimizerRuleParams{}, err
	}
	cooldown := ClampCooldownMinutes(req.CooldownMinutes)
	if req.CooldownMinutes == 0 {
		cooldown = 60
	}
	maxDelta := ClampMaxWeightDeltaPct(req.MaxWeightDeltaPct)

	var campParam, flowParam, brandParam pgtype.UUID
	if strings.TrimSpace(req.CampaignID) != "" {
		campID, err := uuid.Parse(req.CampaignID)
		if err != nil {
			return db.InsertTrafficOptimizerRuleParams{}, fmt.Errorf("invalid campaign_id")
		}
		campParam = domain.ToUUID(campID)
	}
	if strings.TrimSpace(req.FlowID) != "" {
		flowID, err := uuid.Parse(req.FlowID)
		if err != nil {
			return db.InsertTrafficOptimizerRuleParams{}, fmt.Errorf("invalid flow_id")
		}
		flowParam = domain.ToUUID(flowID)
	}
	if strings.TrimSpace(req.BrandID) != "" {
		brandID, err := uuid.Parse(req.BrandID)
		if err != nil {
			return db.InsertTrafficOptimizerRuleParams{}, fmt.Errorf("invalid brand_id")
		}
		brandParam = domain.ToUUID(brandID)
	}
	var presetKey pgtype.Text
	if strings.TrimSpace(req.PresetKey) != "" {
		presetKey = pgtype.Text{String: strings.TrimSpace(req.PresetKey), Valid: true}
	}
	return db.InsertTrafficOptimizerRuleParams{
		CustomerID:          domain.ToUUID(customerID),
		CampaignID:          campParam,
		FlowID:              flowParam,
		BrandID:             brandParam,
		Name:                name,
		Scope:               scope,
		Objective:           objective,
		Algorithm:           algorithm,
		LookbackMinutes:     int32(lookback),
		MinClicks:           int32(minClicks),
		MinConversions:      int32(req.MinConversions),
		MinSpendMicro:       minSpend,
		EvalIntervalMinutes: int32(evalInterval),
		CooldownMinutes:     int32(cooldown),
		MaxWeightDeltaPct:   int32(maxDelta),
		PresetKey:           presetKey,
		PresetParameters:    presetParams,
		Enabled:             req.Enabled,
	}, nil
}

func ruleToDTO(row db.TrafficOptimizerRule) RuleDTO {
	dto := RuleDTO{
		ID:                  uuid.UUID(row.ID.Bytes).String(),
		CustomerID:          uuid.UUID(row.CustomerID.Bytes).String(),
		Name:                row.Name,
		Scope:               row.Scope,
		Objective:           row.Objective,
		Algorithm:           row.Algorithm,
		LookbackMinutes:     int(row.LookbackMinutes),
		MinClicks:           int(row.MinClicks),
		MinConversions:      int(row.MinConversions),
		MinSpendMicro:       row.MinSpendMicro,
		EvalIntervalMinutes: int(row.EvalIntervalMinutes),
		CooldownMinutes:     int(row.CooldownMinutes),
		MaxWeightDeltaPct:   int(row.MaxWeightDeltaPct),
		Enabled:             row.Enabled,
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
	if dto.EvalIntervalMinutes <= 0 {
		dto.EvalIntervalMinutes = 15
	}
	if row.CampaignID.Valid {
		dto.CampaignID = uuid.UUID(row.CampaignID.Bytes).String()
	}
	if row.FlowID.Valid {
		dto.FlowID = uuid.UUID(row.FlowID.Bytes).String()
	}
	if row.BrandID.Valid {
		dto.BrandID = uuid.UUID(row.BrandID.Bytes).String()
	}
	if row.PresetKey.Valid {
		dto.PresetKey = row.PresetKey.String
	}
	if row.LastEvaluatedAt.Valid {
		t := row.LastEvaluatedAt.Time
		dto.LastEvaluatedAt = &t
	}
	return dto
}

func ApplyPreset(req UpsertRuleRequest) (UpsertRuleRequest, error) {
	key := strings.TrimSpace(req.PresetKey)
	if key == "" {
		return req, nil
	}
	expanded, err := ExpandPreset(key, req.PresetParameters)
	if err != nil {
		return req, err
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = expanded.Name
	}
	if strings.TrimSpace(req.Scope) == "" {
		req.Scope = expanded.Scope
	}
	if strings.TrimSpace(req.Objective) == "" {
		req.Objective = expanded.Objective
	}
	if strings.TrimSpace(req.Algorithm) == "" {
		req.Algorithm = expanded.Algorithm
	}
	if req.LookbackMinutes == 0 {
		req.LookbackMinutes = expanded.LookbackMinutes
	}
	if req.MinClicks == 0 {
		req.MinClicks = expanded.MinClicks
	}
	if req.MinConversions == 0 {
		req.MinConversions = expanded.MinConversions
	}
	if req.MinSpendMicro == 0 {
		req.MinSpendMicro = expanded.MinSpendMicro
	}
	if req.EvalIntervalMinutes == 0 {
		req.EvalIntervalMinutes = expanded.EvalIntervalMinutes
	}
	if req.CooldownMinutes == 0 {
		req.CooldownMinutes = expanded.CooldownMinutes
	}
	if req.MaxWeightDeltaPct == 0 {
		req.MaxWeightDeltaPct = expanded.MaxWeightDeltaPct
	}
	return req, nil
}
