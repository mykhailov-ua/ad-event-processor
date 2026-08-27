package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AutomationRuleDTO struct {
	ID              string              `json:"id"`
	CustomerID      string              `json:"customer_id"`
	CampaignID      string              `json:"campaign_id,omitempty"`
	Name            string              `json:"name"`
	Metric          string              `json:"metric"`
	Operator        string              `json:"operator"`
	Threshold       float64             `json:"threshold"`
	WindowMinutes   int                 `json:"window_minutes"`
	GroupBy         string              `json:"group_by"`
	Actions         []automation.Action `json:"actions"`
	CooldownMinutes int                 `json:"cooldown_minutes"`
	Enabled         bool                `json:"enabled"`
	LastFiredAt     *time.Time          `json:"last_fired_at,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type UpsertAutomationRuleRequest struct {
	CustomerID      string              `json:"customer_id"`
	CampaignID      string              `json:"campaign_id"`
	Name            string              `json:"name"`
	Metric          string              `json:"metric"`
	Operator        string              `json:"operator"`
	Threshold       float64             `json:"threshold"`
	WindowMinutes   int                 `json:"window_minutes"`
	GroupBy         string              `json:"group_by"`
	Actions         []automation.Action `json:"actions"`
	CooldownMinutes int                 `json:"cooldown_minutes"`
	Enabled         bool                `json:"enabled"`
}

type AutomationDryRunResponse struct {
	WouldFire []automation.WouldFire `json:"would_fire"`
}

func (s *Service) ListAutomationRules(ctx context.Context, customerID uuid.UUID) ([]AutomationRuleDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(s.pool).ListAutomationRulesByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]AutomationRuleDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := automationRuleToDTO(row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

func (s *Service) CreateAutomationRule(ctx context.Context, req UpsertAutomationRuleRequest) (AutomationRuleDTO, error) {
	if s == nil || s.pool == nil {
		return AutomationRuleDTO{}, fmt.Errorf("service unavailable")
	}
	params, err := buildAutomationRuleParams(req)
	if err != nil {
		return AutomationRuleDTO{}, err
	}
	row, err := db.New(s.pool).InsertAutomationRule(ctx, params)
	if err != nil {
		return AutomationRuleDTO{}, err
	}
	return automationRuleToDTO(row)
}

func (s *Service) UpdateAutomationRule(ctx context.Context, ruleID uuid.UUID, req UpsertAutomationRuleRequest) (AutomationRuleDTO, error) {
	if s == nil || s.pool == nil {
		return AutomationRuleDTO{}, fmt.Errorf("service unavailable")
	}
	params, err := buildAutomationRuleParams(req)
	if err != nil {
		return AutomationRuleDTO{}, err
	}
	row, err := db.New(s.pool).UpdateAutomationRule(ctx, db.UpdateAutomationRuleParams{
		ID:              domain.ToUUID(ruleID),
		Name:            params.Name,
		CampaignID:      params.CampaignID,
		Metric:          params.Metric,
		Operator:        params.Operator,
		Threshold:       params.Threshold,
		WindowMinutes:   params.WindowMinutes,
		GroupBy:         params.GroupBy,
		Actions:         params.Actions,
		CooldownMinutes: params.CooldownMinutes,
		Enabled:         params.Enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AutomationRuleDTO{}, fmt.Errorf("rule not found")
		}
		return AutomationRuleDTO{}, err
	}
	return automationRuleToDTO(row)
}

func (s *Service) DeleteAutomationRule(ctx context.Context, ruleID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	tag, err := db.New(s.pool).DeleteAutomationRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return err
	}
	if tag == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *Service) DryRunAutomationRule(ctx context.Context, ruleID uuid.UUID) (AutomationDryRunResponse, error) {
	if s == nil || s.pool == nil {
		return AutomationDryRunResponse{}, fmt.Errorf("service unavailable")
	}
	row, err := db.New(s.pool).GetAutomationRule(ctx, domain.ToUUID(ruleID))
	if err != nil {
		return AutomationDryRunResponse{}, fmt.Errorf("rule not found")
	}
	rule, err := automation.RuleFromRow(row)
	if err != nil {
		return AutomationDryRunResponse{}, err
	}
	campaignIDs, err := s.resolveAutomationCampaignIDs(ctx, rule)
	if err != nil {
		return AutomationDryRunResponse{}, err
	}
	ch := s.ClickHouseQuery()
	w := automation.NewWorker(s.pool, ch, nil, time.Minute)
	would, err := w.DryRun(ctx, rule, campaignIDs)
	if err != nil {
		return AutomationDryRunResponse{}, err
	}
	return AutomationDryRunResponse{WouldFire: would}, nil
}

func (s *Service) resolveAutomationCampaignIDs(ctx context.Context, rule automation.Rule) ([]uuid.UUID, error) {
	if rule.HasCampaign {
		return []uuid.UUID{rule.CampaignID}, nil
	}
	rows, err := db.New(s.pool).ListCampaignIDsByCustomers(ctx, []pgtype.UUID{domain.ToUUID(rule.CustomerID)})
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, uuid.UUID(row.CampaignID.Bytes))
	}
	return out, nil
}

func buildAutomationRuleParams(req UpsertAutomationRuleRequest) (db.InsertAutomationRuleParams, error) {
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return db.InsertAutomationRuleParams{}, fmt.Errorf("invalid customer_id")
	}
	metric, err := automation.NormalizeMetric(req.Metric)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	operator, err := automation.NormalizeOperator(req.Operator)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	groupBy, err := automation.NormalizeGroupBy(req.GroupBy)
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
	if err := validateAutomationActions(req.Actions, groupBy); err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	actionsRaw, err := automation.MarshalActions(req.Actions)
	if err != nil {
		return db.InsertAutomationRuleParams{}, err
	}
	window := automation.ClampWindowMinutes(req.WindowMinutes)
	if req.WindowMinutes == 0 {
		window = 60
	}
	cooldown := automation.ClampCooldownMinutes(req.CooldownMinutes)
	if req.CooldownMinutes == 0 {
		cooldown = 60
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
		CustomerID:      domain.ToUUID(customerID),
		CampaignID:      campParam,
		Name:            name,
		Metric:          metric,
		Operator:        operator,
		Threshold:       req.Threshold,
		WindowMinutes:   int32(window),
		GroupBy:         groupBy,
		Actions:         actionsRaw,
		CooldownMinutes: int32(cooldown),
		Enabled:         req.Enabled,
	}, nil
}

func validateAutomationActions(actions []automation.Action, groupBy string) error {
	for _, action := range actions {
		switch action.Type {
		case automation.ActionNotify:
			if !strings.HasPrefix(strings.TrimSpace(action.WebhookURL), "http") {
				return fmt.Errorf("notify action requires webhook_url")
			}
		case automation.ActionPauseCampaign, automation.ActionBlacklistPlacement, automation.ActionPlatformPause:
			if action.Type == automation.ActionBlacklistPlacement && groupBy != automation.GroupByPlacement {
				return fmt.Errorf("blacklist_placement requires group_by placement_id")
			}
			if action.Type == automation.ActionPlatformPause && strings.TrimSpace(action.Network) == "" {
				return fmt.Errorf("platform_pause requires network")
			}
		default:
			return fmt.Errorf("unsupported action %q", action.Type)
		}
	}
	return nil
}

func automationRuleToDTO(row db.AutomationRule) (AutomationRuleDTO, error) {
	actions, err := automation.ParseActions(row.Actions)
	if err != nil {
		return AutomationRuleDTO{}, err
	}
	dto := AutomationRuleDTO{
		ID:              uuid.UUID(row.ID.Bytes).String(),
		CustomerID:      uuid.UUID(row.CustomerID.Bytes).String(),
		Name:            row.Name,
		Metric:          row.Metric,
		Operator:        row.Operator,
		Threshold:       row.Threshold,
		WindowMinutes:   int(row.WindowMinutes),
		GroupBy:         row.GroupBy,
		Actions:         actions,
		CooldownMinutes: int(row.CooldownMinutes),
		Enabled:         row.Enabled,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
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
