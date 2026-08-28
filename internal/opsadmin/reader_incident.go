package opsadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

func (r *Reader) incidentDashboardStale(ctx context.Context) (bool, int) {
	if r == nil || r.pool() == nil || r.deps.ClickHouseQuery == nil {
		return true, 0
	}
	lag, err := r.deps.ClickHouseQuery.IngestionLag(ctx)
	if err != nil {
		return true, 0
	}
	stale, lagSec := database.Freshness(lag, 5*time.Minute)
	return stale, lagSec
}

func (r *Reader) listAffectedCampaigns(ctx context.Context, limit int) ([]AffectedCampaignDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool().Query(ctx, `
		SELECT id, name
		FROM campaigns
		WHERE deleted_at IS NULL AND status = 'active'
		ORDER BY updated_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AffectedCampaignDTO
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, AffectedCampaignDTO{
			CampaignID: id.String(),
			Name:       name,
		})
	}
	return out, rows.Err()
}
