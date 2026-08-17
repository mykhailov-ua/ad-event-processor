package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
)

func (r *opsReader) incidentDashboardStale(ctx context.Context) (bool, int) {
	if r == nil || r.svc == nil || r.svc.CHQuery() == nil {
		return true, 0
	}
	lag, err := r.svc.CHQuery().IngestionLag(ctx)
	if err != nil {
		return true, 0
	}
	stale, lagSec := database.Freshness(lag, 5*time.Minute)
	return stale, lagSec
}

func (r *opsReader) listAffectedCampaigns(ctx context.Context, limit int) ([]adminapi.AffectedCampaignDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT id, name
		FROM campaigns
		WHERE deleted_at IS NULL AND status = 'active'
		ORDER BY updated_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []adminapi.AffectedCampaignDTO
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, adminapi.AffectedCampaignDTO{
			CampaignID: id.String(),
			Name:       name,
		})
	}
	return out, rows.Err()
}
