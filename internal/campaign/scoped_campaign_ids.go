package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/teamscope"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListScopedCampaignIDs returns campaign IDs visible to the caller for customerID.
// Media buyers and enforced masked buyers see owned campaigns; TL sees team subset.
func ListScopedCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	return teamscope.ScopedCampaignIDsQuery(ctx, pool, customerID)
}

// SessionScopedCampaignIDs returns (allCampaigns=true, nil, nil) for unscoped operators.
// Bound sessions return scoped IDs for the session customer (empty slice when none visible).
func SessionScopedCampaignIDs(ctx context.Context, pool *pgxpool.Pool) (allCampaigns bool, ids []uuid.UUID, err error) {
	u, ok := authz.GetUser(ctx)
	if !ok || !u.HasBoundCustomer() {
		return true, nil, nil
	}
	ids, err = ListScopedCampaignIDs(ctx, pool, u.CustomerID)
	return false, ids, err
}
