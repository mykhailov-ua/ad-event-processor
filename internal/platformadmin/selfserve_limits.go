package platformadmin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SelfServeLimitsHost interface {
	Pool() *pgxpool.Pool
	SelfServeBudgetMinMicro() int64
	SelfServeBudgetMaxMicro() int64
	SelfServeMaxActiveCampaigns() int
	SelfServeMaxCreatesPerDay() int
}

func EnforceSelfServeCreateLimits(ctx context.Context, host SelfServeLimitsHost, customerID uuid.UUID, budgetMicro int64) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	if host.SelfServeBudgetMinMicro() > 0 && budgetMicro < host.SelfServeBudgetMinMicro() {
		return fmt.Errorf("%w: minimum %d micro", ErrSelfServeBudgetOutOfRange, host.SelfServeBudgetMinMicro())
	}
	if host.SelfServeBudgetMaxMicro() > 0 && budgetMicro > host.SelfServeBudgetMaxMicro() {
		return fmt.Errorf("%w: maximum %d micro", ErrSelfServeBudgetOutOfRange, host.SelfServeBudgetMaxMicro())
	}

	var active int64
	err := host.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE customer_id = $1 AND status = 'ACTIVE'`, customerID).Scan(&active)
	if err != nil {
		return fmt.Errorf("count active campaigns: %w", err)
	}
	if host.SelfServeMaxActiveCampaigns() > 0 && int(active) >= host.SelfServeMaxActiveCampaigns() {
		return ErrSelfServeActiveCampaignLimit
	}

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var createdToday int64
	err = host.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE customer_id = $1 AND created_at >= $2`, customerID, startOfDay).Scan(&createdToday)
	if err != nil {
		return fmt.Errorf("count daily campaign creates: %w", err)
	}
	if host.SelfServeMaxCreatesPerDay() > 0 && int(createdToday) >= host.SelfServeMaxCreatesPerDay() {
		return ErrSelfServeDailyCreateLimit
	}
	return nil
}
