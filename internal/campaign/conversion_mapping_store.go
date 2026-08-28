package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ListCampaignConversionMappings(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) ([]ConversionMappingDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	rows, err := db.New(pool).ListConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]ConversionMappingDTO, 0, len(rows))
	for i := range rows {
		out = append(out, ConversionMappingToDTO(&rows[i]))
	}
	return out, nil
}

func ReplaceCampaignConversionMappings(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeConversionMappings(mappings)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	if err := q.DeleteConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return nil, err
	}
	for i := range normalized {
		row := &normalized[i]
		if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
			CampaignID:    domain.ToUUID(campaignID),
			InboundStatus: row.InboundStatus,
			GoalName:      row.GoalName,
			PayoutMicro:   row.PayoutMicro,
		}); err != nil {
			return nil, fmt.Errorf("insert conversion mapping: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return normalized, nil
}
