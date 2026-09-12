package campaign

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusSchemePayoutInherit     = "inherit"
	StatusSchemePayoutFixed       = "fixed"
	StatusSchemePayoutZero        = "zero"
	StatusSchemePayoutPassThrough = "pass_through"
	StatusSchemePayoutAccumulate  = "accumulate_payout"
)

type StatusSchemeRuleDTO struct {
	ID                string `json:"id"`
	SortOrder         int32  `json:"sort_order"`
	WhenStatus        string `json:"when_status"`
	WhenGoal          string `json:"when_goal"`
	SetInternalStatus string `json:"set_internal_status"`
	SetGoalName       string `json:"set_goal_name"`
	PayoutMode        string `json:"payout_mode"`
	PayoutMicro       int64  `json:"payout_micro"`
	FireOutbound      bool   `json:"fire_outbound"`
	Enabled           bool   `json:"enabled"`
}

type StatusSchemeListResponse struct {
	Rules []StatusSchemeRuleDTO `json:"rules"`
}

type ReplaceStatusSchemeRequest struct {
	Rules []StatusSchemeRuleDTO `json:"rules"`
}

type PatchStatusSchemeRuleRequest struct {
	WhenStatus        *string `json:"when_status,omitempty"`
	WhenGoal          *string `json:"when_goal,omitempty"`
	SetInternalStatus *string `json:"set_internal_status,omitempty"`
	SetGoalName       *string `json:"set_goal_name,omitempty"`
	PayoutMode        *string `json:"payout_mode,omitempty"`
	PayoutMicro       *int64  `json:"payout_micro,omitempty"`
	FireOutbound      *bool   `json:"fire_outbound,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
	SortOrder         *int32  `json:"sort_order,omitempty"`
}

type StatusSchemeService interface {
	ListCampaignStatusSchemeRules(ctx context.Context, campaignID uuid.UUID) ([]StatusSchemeRuleDTO, error)
	ReplaceCampaignStatusSchemeRules(ctx context.Context, campaignID uuid.UUID, rules []StatusSchemeRuleDTO) ([]StatusSchemeRuleDTO, error)
	PatchCampaignStatusSchemeRule(ctx context.Context, campaignID, ruleID uuid.UUID, patch PatchStatusSchemeRuleRequest) (StatusSchemeRuleDTO, error)
}

func StatusSchemeRuleToDTO(row *db.CampaignStatusSchemeRule) StatusSchemeRuleDTO {
	if row == nil {
		return StatusSchemeRuleDTO{}
	}
	return StatusSchemeRuleDTO{
		ID:                uuid.UUID(row.ID.Bytes).String(),
		SortOrder:         row.SortOrder,
		WhenStatus:        row.WhenStatus,
		WhenGoal:          row.WhenGoal,
		SetInternalStatus: row.SetInternalStatus,
		SetGoalName:       row.SetGoalName,
		PayoutMode:        row.PayoutMode,
		PayoutMicro:       row.PayoutMicro,
		FireOutbound:      row.FireOutbound,
		Enabled:           row.Enabled,
	}
}

func NormalizeStatusSchemeRules(rules []StatusSchemeRuleDTO) ([]StatusSchemeRuleDTO, error) {
	if len(rules) == 0 {
		return []StatusSchemeRuleDTO{}, nil
	}
	out := make([]StatusSchemeRuleDTO, 0, len(rules))
	for i := range rules {
		row := rules[i]
		sortOrder := row.SortOrder
		if sortOrder <= 0 {
			sortOrder = int32(i + 1)
		}
		whenStatus := strings.ToLower(strings.TrimSpace(row.WhenStatus))
		whenGoal := strings.ToLower(strings.TrimSpace(row.WhenGoal))
		if whenStatus == "" && whenGoal == "" {
			return nil, fmt.Errorf("rule %d: when_status or when_goal is required", i+1)
		}
		payoutMode := strings.ToLower(strings.TrimSpace(row.PayoutMode))
		if payoutMode == "" {
			payoutMode = StatusSchemePayoutInherit
		}
		switch payoutMode {
		case StatusSchemePayoutInherit, StatusSchemePayoutFixed, StatusSchemePayoutZero, StatusSchemePayoutPassThrough, StatusSchemePayoutAccumulate:
		default:
			return nil, fmt.Errorf("rule %d: invalid payout_mode", i+1)
		}
		if payoutMode == StatusSchemePayoutFixed && row.PayoutMicro < 0 {
			return nil, fmt.Errorf("rule %d: payout_micro must be non-negative", i+1)
		}
		out = append(out, StatusSchemeRuleDTO{
			SortOrder:         sortOrder,
			WhenStatus:        whenStatus,
			WhenGoal:          whenGoal,
			SetInternalStatus: strings.TrimSpace(row.SetInternalStatus),
			SetGoalName:       strings.TrimSpace(row.SetGoalName),
			PayoutMode:        payoutMode,
			PayoutMicro:       row.PayoutMicro,
			FireOutbound:      row.FireOutbound,
			Enabled:           row.Enabled,
		})
	}
	return out, nil
}

func ListCampaignStatusSchemeRules(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) ([]StatusSchemeRuleDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	rows, err := db.New(pool).ListStatusSchemeRulesByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]StatusSchemeRuleDTO, 0, len(rows))
	for i := range rows {
		out = append(out, StatusSchemeRuleToDTO(&rows[i]))
	}
	return out, nil
}

func ReplaceCampaignStatusSchemeRules(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, rules []StatusSchemeRuleDTO) ([]StatusSchemeRuleDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeStatusSchemeRules(rules)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	saved, err := ReplaceCampaignStatusSchemeRulesTx(ctx, db.New(tx), campaignID, normalized)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func ReplaceCampaignStatusSchemeRulesTx(
	ctx context.Context,
	q *db.Queries,
	campaignID uuid.UUID,
	rules []StatusSchemeRuleDTO,
) ([]StatusSchemeRuleDTO, error) {
	if q == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeStatusSchemeRules(rules)
	if err != nil {
		return nil, err
	}
	if err := q.DeleteStatusSchemeRulesByCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return nil, err
	}
	out := make([]StatusSchemeRuleDTO, 0, len(normalized))
	for i := range normalized {
		row := normalized[i]
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		inserted, err := q.InsertStatusSchemeRule(ctx, db.InsertStatusSchemeRuleParams{
			ID:                domain.ToUUID(id),
			CampaignID:        domain.ToUUID(campaignID),
			SortOrder:         row.SortOrder,
			WhenStatus:        row.WhenStatus,
			WhenGoal:          row.WhenGoal,
			SetInternalStatus: row.SetInternalStatus,
			SetGoalName:       row.SetGoalName,
			PayoutMode:        row.PayoutMode,
			PayoutMicro:       row.PayoutMicro,
			FireOutbound:      row.FireOutbound,
			Enabled:           row.Enabled,
		})
		if err != nil {
			return nil, fmt.Errorf("insert status scheme rule: %w", err)
		}
		out = append(out, StatusSchemeRuleToDTO(&inserted))
	}
	return out, nil
}

func PatchCampaignStatusSchemeRule(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID, ruleID uuid.UUID,
	patch PatchStatusSchemeRuleRequest,
) (StatusSchemeRuleDTO, error) {
	if pool == nil {
		return StatusSchemeRuleDTO{}, errServiceUnavailable()
	}
	q := db.New(pool)
	current, err := q.GetStatusSchemeRule(ctx, db.GetStatusSchemeRuleParams{
		ID:         domain.ToUUID(ruleID),
		CampaignID: domain.ToUUID(campaignID),
	})
	if err != nil {
		return StatusSchemeRuleDTO{}, err
	}
	next := StatusSchemeRuleToDTO(&current)
	if patch.WhenStatus != nil {
		next.WhenStatus = strings.ToLower(strings.TrimSpace(*patch.WhenStatus))
	}
	if patch.WhenGoal != nil {
		next.WhenGoal = strings.ToLower(strings.TrimSpace(*patch.WhenGoal))
	}
	if patch.SetInternalStatus != nil {
		next.SetInternalStatus = strings.TrimSpace(*patch.SetInternalStatus)
	}
	if patch.SetGoalName != nil {
		next.SetGoalName = strings.TrimSpace(*patch.SetGoalName)
	}
	if patch.PayoutMode != nil {
		next.PayoutMode = strings.ToLower(strings.TrimSpace(*patch.PayoutMode))
	}
	if patch.PayoutMicro != nil {
		next.PayoutMicro = *patch.PayoutMicro
	}
	if patch.FireOutbound != nil {
		next.FireOutbound = *patch.FireOutbound
	}
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if patch.SortOrder != nil {
		next.SortOrder = *patch.SortOrder
	}
	normalized, err := NormalizeStatusSchemeRules([]StatusSchemeRuleDTO{next})
	if err != nil {
		return StatusSchemeRuleDTO{}, err
	}
	next = normalized[0]
	updated, err := q.UpdateStatusSchemeRule(ctx, db.UpdateStatusSchemeRuleParams{
		ID:                domain.ToUUID(ruleID),
		CampaignID:        domain.ToUUID(campaignID),
		WhenStatus:        next.WhenStatus,
		WhenGoal:          next.WhenGoal,
		SetInternalStatus: next.SetInternalStatus,
		SetGoalName:       next.SetGoalName,
		PayoutMode:        next.PayoutMode,
		PayoutMicro:       next.PayoutMicro,
		FireOutbound:      next.FireOutbound,
		Enabled:           next.Enabled,
	})
	if err != nil {
		return StatusSchemeRuleDTO{}, err
	}
	return StatusSchemeRuleToDTO(&updated), nil
}

func CloneCampaignStatusSchemeRules(ctx context.Context, tx pgx.Tx, destID, sourceID uuid.UUID) error {
	if tx == nil {
		return errServiceUnavailable()
	}
	_, err := tx.Exec(ctx, `
INSERT INTO campaign_status_scheme_rules (
    id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
    payout_mode, payout_micro, fire_outbound, enabled
)
SELECT gen_random_uuid(), $1, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
       payout_mode, payout_micro, fire_outbound, enabled
FROM campaign_status_scheme_rules
WHERE campaign_id = $2`, destID, sourceID)
	return err
}
