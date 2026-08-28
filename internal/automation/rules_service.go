package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleDTO struct {
	ID                  string     `json:"id"`
	CustomerID          string     `json:"customer_id"`
	CampaignID          string     `json:"campaign_id,omitempty"`
	Name                string     `json:"name"`
	Metric              string     `json:"metric"`
	Operator            string     `json:"operator"`
	Threshold           float64    `json:"threshold"`
	WindowMinutes       int        `json:"window_minutes"`
	GroupBy             string     `json:"group_by"`
	Actions             []Action   `json:"actions"`
	CooldownMinutes     int        `json:"cooldown_minutes"`
	EvalIntervalMinutes int        `json:"eval_interval_minutes"`
	Enabled             bool       `json:"enabled"`
	LastFiredAt         *time.Time `json:"last_fired_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type UpsertRuleRequest struct {
	CustomerID          string             `json:"customer_id"`
	CampaignID          string             `json:"campaign_id"`
	Name                string             `json:"name"`
	Metric              string             `json:"metric"`
	Operator            string             `json:"operator"`
	Threshold           float64            `json:"threshold"`
	WindowMinutes       int                `json:"window_minutes"`
	GroupBy             string             `json:"group_by"`
	Actions             []Action           `json:"actions"`
	CooldownMinutes     int                `json:"cooldown_minutes"`
	EvalIntervalMinutes int                `json:"eval_interval_minutes"`
	Enabled             bool               `json:"enabled"`
	PresetKey           string             `json:"preset_key,omitempty"`
	PresetParameters    map[string]float64 `json:"preset_parameters,omitempty"`
}

type DryRunResponse struct {
	WouldFire []WouldFire `json:"would_fire"`
}

type LicenseGate interface {
	ValidateAutomationLicense(ctx context.Context, actions []Action) error
}

type RulesService struct {
	Pool             *pgxpool.Pool
	ClickHouse       *database.ClickHouseQuery
	EvalFloorMinutes func() int
	LicenseGate      LicenseGate
}

func (s *RulesService) ListRules(ctx context.Context, customerID uuid.UUID) ([]RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(s.Pool).ListAutomationRulesByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]RuleDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := ruleToDTO(row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

func (s *RulesService) CreateRule(ctx context.Context, req UpsertRuleRequest) (RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return RuleDTO{}, fmt.Errorf("service unavailable")
	}
	req, err := ApplyPreset(req)
	if err != nil {
		return RuleDTO{}, err
	}
	params, err := s.buildRuleParams(ctx, req)
	if err != nil {
		return RuleDTO{}, err
	}
	row, err := db.New(s.Pool).InsertAutomationRule(ctx, params)
	if err != nil {
		return RuleDTO{}, err
	}
	return ruleToDTO(row)
}

func (s *RulesService) UpdateRule(ctx context.Context, ruleID uuid.UUID, req UpsertRuleRequest) (RuleDTO, error) {
	if s == nil || s.Pool == nil {
		return RuleDTO{}, fmt.Errorf("service unavailable")
	}
	req, err := ApplyPreset(req)
	if err != nil {
		return RuleDTO{}, err
	}
	params, err := s.buildRuleParams(ctx, req)
	if err != nil {
		return RuleDTO{}, err
	}
	row, err := db.New(s.Pool).UpdateAutomationRule(ctx, db.UpdateAutomationRuleParams{
		ID:                  domain.ToUUID(ruleID),
		Name:                params.Name,
		CampaignID:          params.CampaignID,
		Metric:              params.Metric,
		Operator:            params.Operator,
		Threshold:           params.Threshold,
		WindowMinutes:       params.WindowMinutes,
		GroupBy:             params.GroupBy,
		Actions:             params.Actions,
		CooldownMinutes:     params.CooldownMinutes,
		EvalIntervalMinutes: params.EvalIntervalMinutes,
		Enabled:             params.Enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuleDTO{}, fmt.Errorf("rule not found")
		}
		return RuleDTO{}, err
	}
	return ruleToDTO(row)
}

func (s *RulesService) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	if s == nil || s.Pool == nil {
		return fmt.Errorf("service unavailable")
	}
	tag, err := db.New(s.Pool).DeleteAutomationRule(ctx, domain.ToUUID(ruleID))
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
		return DryRunResponse{}, fmt.Errorf("service unavailable")
	}
	row, err := db.New(s.Pool).GetAutomationRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return DryRunResponse{}, fmt.Errorf("rule not found")
	}
	rule, err := RuleFromRow(row)
	if err != nil {
		return DryRunResponse{}, err
	}
	campaignIDs, err := s.resolveCampaignIDs(ctx, rule)
	if err != nil {
		return DryRunResponse{}, err
	}
	w := NewWorker(s.Pool, s.ClickHouse, nil, time.Minute, 50)
	would, err := w.DryRun(ctx, rule, campaignIDs)
	if err != nil {
		return DryRunResponse{}, err
	}
	return DryRunResponse{WouldFire: would}, nil
}

func (s *RulesService) BuildRuleParams(ctx context.Context, req UpsertRuleRequest) (db.InsertAutomationRuleParams, error) {
	return s.buildRuleParams(ctx, req)
}

func (s *RulesService) resolveCampaignIDs(ctx context.Context, rule Rule) ([]uuid.UUID, error) {
	if rule.HasCampaign {
		return []uuid.UUID{rule.CampaignID}, nil
	}
	rows, err := db.New(s.Pool).ListCampaignIDsByCustomers(ctx, []pgtype.UUID{domain.ToUUID(rule.CustomerID)})
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, uuid.UUID(row.CampaignID.Bytes))
	}
	return out, nil
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
	if strings.TrimSpace(req.Metric) == "" {
		req.Metric = expanded.Metric
	}
	if strings.TrimSpace(req.Operator) == "" {
		req.Operator = expanded.Operator
	}
	if req.Threshold == 0 {
		req.Threshold = expanded.Threshold
	}
	if req.WindowMinutes == 0 {
		req.WindowMinutes = expanded.WindowMinutes
	}
	if strings.TrimSpace(req.GroupBy) == "" {
		req.GroupBy = expanded.GroupBy
	}
	if len(req.Actions) == 0 {
		req.Actions = expanded.Actions
	}
	if req.CooldownMinutes == 0 {
		req.CooldownMinutes = expanded.CooldownMinutes
	}
	if req.EvalIntervalMinutes == 0 {
		req.EvalIntervalMinutes = expanded.EvalIntervalMinutes
	}
	return req, nil
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

func (s *RulesService) buildRuleParams(ctx context.Context, req UpsertRuleRequest) (db.InsertAutomationRuleParams, error) {
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return db.InsertAutomationRuleParams{}, fmt.Errorf("invalid customer_id")
	}
	metric, err := NormalizeMetric(req.Metric)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	operator, err := NormalizeOperator(req.Operator)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	groupBy, err := NormalizeGroupBy(req.GroupBy)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return db.InsertAutomationRuleParams{}, fmt.Errorf("name is required")
	}
	if len(req.Actions) == 0 {
		return db.InsertAutomationRuleParams{}, fmt.Errorf("actions is required")
	}
	if err := validateActions(req.Actions, groupBy); err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	if s.LicenseGate != nil {
		if err := s.LicenseGate.ValidateAutomationLicense(ctx, req.Actions); err != nil {
			return db.InsertAutomationRuleParams{}, err
		}
	}
	actionsRaw, err := MarshalActions(req.Actions)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	window := ClampWindowMinutes(req.WindowMinutes)
	if req.WindowMinutes == 0 {
		window = 60
	}
	cooldown := ClampCooldownMinutes(req.CooldownMinutes)
	if req.CooldownMinutes == 0 {
		cooldown = 60
	}
	evalInterval, err := NormalizeEvalIntervalMinutes(req.EvalIntervalMinutes, s.evalFloorMinutes())
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	var campParam pgtype.UUID
	if strings.TrimSpace(req.CampaignID) != "" {
		campID, err := uuid.Parse(req.CampaignID)
		if err != nil {
			return db.InsertAutomationRuleParams{}, fmt.Errorf("invalid campaign_id")
		}
		campParam = domain.ToUUID(campID)
	}
	return db.InsertAutomationRuleParams{
		CustomerID:          domain.ToUUID(customerID),
		CampaignID:          campParam,
		Name:                name,
		Metric:              metric,
		Operator:            operator,
		Threshold:           req.Threshold,
		WindowMinutes:       int32(window),
		GroupBy:             groupBy,
		Actions:             actionsRaw,
		CooldownMinutes:     int32(cooldown),
		EvalIntervalMinutes: int32(evalInterval),
		Enabled:             req.Enabled,
	}, nil
}

func validateActions(actions []Action, groupBy string) error {
	for _, action := range actions {
		switch action.Type {
		case ActionNotify:
			if !strings.HasPrefix(strings.TrimSpace(action.WebhookURL), "http") {
				return fmt.Errorf("notify action requires webhook_url")
			}
		case ActionPauseCampaign, ActionBlacklistPlacement, ActionPlatformPause:
			if action.Type == ActionBlacklistPlacement && groupBy != GroupByPlacement {
				return fmt.Errorf("blacklist_placement requires group_by placement_id")
			}
			if action.Type == ActionPlatformPause && strings.TrimSpace(action.Network) == "" {
				return fmt.Errorf("platform_pause requires network")
			}
		default:
			return fmt.Errorf("unsupported action %q", action.Type)
		}
	}
	return nil
}

func ruleToDTO(row db.AutomationRule) (RuleDTO, error) {
	actions, err := ParseActions(row.Actions)
	if err != nil {
		return RuleDTO{}, err
	}
	dto := RuleDTO{
		ID:                  uuid.UUID(row.ID.Bytes).String(),
		CustomerID:          uuid.UUID(row.CustomerID.Bytes).String(),
		Name:                row.Name,
		Metric:              row.Metric,
		Operator:            row.Operator,
		Threshold:           row.Threshold,
		WindowMinutes:       int(row.WindowMinutes),
		GroupBy:             row.GroupBy,
		Actions:             actions,
		CooldownMinutes:     int(row.CooldownMinutes),
		EvalIntervalMinutes: int(row.EvalIntervalMinutes),
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
	if row.LastFiredAt.Valid {
		t := row.LastFiredAt.Time
		dto.LastFiredAt = &t
	}
	return dto, nil
}
