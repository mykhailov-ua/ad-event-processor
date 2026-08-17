package controlplane

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBudgetApprovalRequired = errors.New("budget approval required")

// memberBudgetAllocationMicro sums budget_limit for campaigns owned by userID (optionally replacing one campaign limit).
func memberBudgetAllocationMicro(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	replaceCampaignID uuid.UUID,
	replaceWithLimit int64,
) (int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, budget_limit
		FROM campaigns
		WHERE owner_user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total int64
	for rows.Next() {
		var campID uuid.UUID
		var limit int64
		if err := rows.Scan(&campID, &limit); err != nil {
			return 0, err
		}
		if campID == replaceCampaignID {
			total += replaceWithLimit
		} else {
			total += limit
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return total, nil
}

func memberSpendCapMicro(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, bool, error) {
	var spendCap int64
	err := pool.QueryRow(ctx, `
		SELECT spend_cap_micro FROM team_member_limits WHERE user_id = $1`, userID).Scan(&spendCap)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return spendCap, true, nil
}

// checkMediaBuyerBudgetCap returns ErrBudgetApprovalRequired when a budget increase exceeds the member cap.
func (s *Service) checkMediaBuyerBudgetCap(
	ctx context.Context,
	userID uuid.UUID,
	campaignID uuid.UUID,
	newLimit int64,
) error {
	if s == nil || s.GetPool() == nil {
		return nil
	}
	spendCap, hasCap, err := memberSpendCapMicro(ctx, s.GetPool(), userID)
	if err != nil || !hasCap || spendCap <= 0 {
		return err
	}
	total, err := memberBudgetAllocationMicro(ctx, s.GetPool(), userID, campaignID, newLimit)
	if err != nil {
		return err
	}
	if total > spendCap {
		return ErrBudgetApprovalRequired
	}
	return nil
}
