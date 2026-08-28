package governance

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MemberBudgetAllocationMicro(
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

func MemberSpendCapMicro(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, bool, error) {
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

func CheckMediaBuyerBudgetCap(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	campaignID uuid.UUID,
	newLimit int64,
) error {
	if pool == nil {
		return nil
	}
	spendCap, hasCap, err := MemberSpendCapMicro(ctx, pool, userID)
	if err != nil || !hasCap || spendCap <= 0 {
		return err
	}
	total, err := MemberBudgetAllocationMicro(ctx, pool, userID, campaignID, newLimit)
	if err != nil {
		return err
	}
	if total > spendCap {
		return ErrBudgetApprovalRequired
	}
	return nil
}
