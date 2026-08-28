package controlplane

import (
	"context"
	"errors"

	db 	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/governance"

	"github.com/google/uuid"
)

type auditBudgetApprovalAutoDeny struct {
	CampaignID           string `json:"campaign_id"`
	RequestedBudgetMicro int64  `json:"requested_budget_micro"`
	PreviousBudgetMicro  int64  `json:"previous_budget_micro"`
	Reason               string `json:"reason"`
}

type campaignBudgetApprovalState struct {
	PreviousBudgetMicro  int64
	RequestedBudgetMicro int64
}

func (s *Service) evaluateBudgetApprovalAction(
	ctx context.Context,
	locked db.Campaign,
	userID uuid.UUID,
	newLimit int64,
) (string, string, error) {
	if s == nil || s.GetPool() == nil {
		return "pending", "", nil
	}
	spendCap, hasCap, err := governance.MemberSpendCapMicro(ctx, s.GetPool(), userID)
	if err != nil {
		return "", "", err
	}
	if !hasCap || spendCap <= 0 {
		return "pending", "", nil
	}
	campaignID := uuid.UUID(locked.ID.Bytes)
	total, err := governance.MemberBudgetAllocationMicro(ctx, s.GetPool(), userID, campaignID, newLimit)
	if err != nil {
		return "", "", err
	}
	if total <= spendCap {
		return "allow", "", nil
	}
	if newLimit > spendCap {
		return "auto_denied", "exceeds_team_spend_cap", nil
	}
	delta := newLimit - locked.BudgetLimit
	if delta > 0 {
		cust, err := db.New(s.GetPool()).GetCustomerForUpdate(ctx, locked.CustomerID)
		if err != nil {
			return "", "", mapNotFound(err, ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < delta {
			return "auto_denied", "insufficient_customer_balance", nil
		}
	}
	return "pending", "", nil
}

func (s *Service) insertBudgetApproval(
	ctx context.Context,
	customerID, userID, campaignID uuid.UUID,
	previousLimit, requestedLimit int64,
	status string,
) (uuid.UUID, error) {
	pool := s.GetPool()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO team_budget_approvals (
			id, customer_id, user_id, campaign_id,
			requested_budget_micro, previous_budget_micro, status, resolved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, CASE WHEN $7 <> 'PENDING' THEN NOW() ELSE NULL END)`,
		id, customerID, userID, campaignID, requestedLimit, previousLimit, status)
	return id, err
}

func (s *Service) handleMediaBuyerBudgetIncrease(
	ctx context.Context,
	locked db.Campaign,
	userID uuid.UUID,
	newLimit int64,
) error {
	campaignID := uuid.UUID(locked.ID.Bytes)
	customerID := uuid.UUID(locked.CustomerID.Bytes)
	if err := s.checkMediaBuyerBudgetCap(ctx, userID, campaignID, newLimit); err == nil {
		return nil
	} else if !errors.Is(err, ErrBudgetApprovalRequired) {
		return err
	}
	action, reason, err := s.evaluateBudgetApprovalAction(ctx, locked, userID, newLimit)
	if err != nil {
		return err
	}
	switch action {
	case "allow":
		return nil
	case "auto_denied":
		approvalID, createErr := s.insertBudgetApproval(ctx, customerID, userID, campaignID, locked.BudgetLimit, newLimit, "DENIED")
		if createErr != nil {
			return createErr
		}
		s.AuditLog(ctx, nil, userID, "AUTO_DENY_BUDGET_APPROVAL", "team_budget_approval", &approvalID, auditBudgetApprovalAutoDeny{
			CampaignID:           campaignID.String(),
			RequestedBudgetMicro: newLimit,
			PreviousBudgetMicro:  locked.BudgetLimit,
			Reason:               reason,
		}, nil)
		return ErrBudgetApprovalAutoDenied
	default:
		if _, createErr := s.insertBudgetApproval(ctx, customerID, userID, campaignID, locked.BudgetLimit, newLimit, "PENDING"); createErr != nil {
			return createErr
		}
		return ErrBudgetApprovalRequired
	}
}

func (s *Service) pendingBudgetApprovalsByCampaign(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]campaignBudgetApprovalState, error) {
	out := make(map[uuid.UUID]campaignBudgetApprovalState)
	if s == nil || s.GetPool() == nil || len(campaignIDs) == 0 {
		return out, nil
	}
	rows, err := s.GetPool().Query(ctx, `
		SELECT campaign_id, previous_budget_micro, requested_budget_micro
		FROM team_budget_approvals
		WHERE status = 'PENDING' AND campaign_id = ANY($1)`, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var campaignID uuid.UUID
		var state campaignBudgetApprovalState
		if err := rows.Scan(&campaignID, &state.PreviousBudgetMicro, &state.RequestedBudgetMicro); err != nil {
			return nil, err
		}
		out[campaignID] = state
	}
	return out, rows.Err()
}

func (s *Service) attachCampaignBudgetApprovalState(ctx context.Context, dto *CampaignDTO) {
	if dto == nil || s == nil {
		return
	}
	campaignID, err := uuid.Parse(dto.ID)
	if err != nil {
		return
	}
	states, err := s.pendingBudgetApprovalsByCampaign(ctx, []uuid.UUID{campaignID})
	if err != nil || len(states) == 0 {
		return
	}
	state, ok := states[campaignID]
	if !ok {
		return
	}
	effective := state.PreviousBudgetMicro
	pending := state.RequestedBudgetMicro
	dto.EffectiveBudgetMicros = &effective
	dto.PendingBudgetMicros = &pending
}

func (s *Service) attachCampaignListBudgetApprovalStates(ctx context.Context, items []CampaignDTO) {
	if s == nil || len(items) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(items))
	for i := range items {
		campaignID, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		ids = append(ids, campaignID)
	}
	states, err := s.pendingBudgetApprovalsByCampaign(ctx, ids)
	if err != nil {
		return
	}
	for i := range items {
		campaignID, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		state, ok := states[campaignID]
		if !ok {
			continue
		}
		effective := state.PreviousBudgetMicro
		pending := state.RequestedBudgetMicro
		items[i].EffectiveBudgetMicros = &effective
		items[i].PendingBudgetMicros = &pending
	}
}
