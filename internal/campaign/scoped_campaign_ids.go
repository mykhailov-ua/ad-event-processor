package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListScopedCampaignIDs returns campaign IDs visible to the caller for customerID.
// Media buyers are limited to campaigns they own; other bound roles see all campaigns for the customer.
func ListScopedCampaignIDs(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	if pool == nil || customerID == uuid.Nil {
		return nil, nil
	}
	ownerFilter := CampaignOwnerUserFilter(ctx)
	var (
		rows pgx.Rows
		err  error
	)
	if ownerFilter.Valid {
		rows, err = pool.Query(ctx, `
			SELECT id FROM campaigns
			WHERE customer_id = $1 AND deleted_at IS NULL AND owner_user_id = $2`,
			customerID, ownerFilter)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT id FROM campaigns
			WHERE customer_id = $1 AND deleted_at IS NULL`,
			customerID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCampaignIDRows(rows)
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

func scanCampaignIDRows(rows pgx.Rows) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
