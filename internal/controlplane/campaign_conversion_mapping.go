package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ConversionMappingListResponse = campaign.ConversionMappingListResponse

type ReplaceConversionMappingsRequest = campaign.ReplaceConversionMappingsRequest

func (s *Service) ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]ConversionMappingDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(s.pool).ListConversionMappingsByCampaign(ctx, domainToPgUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]ConversionMappingDTO, 0, len(rows))
	for i := range rows {
		out = append(out, conversionMappingToDTO(&rows[i]))
	}
	return out, nil
}

func (s *Service) ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	normalized, err := normalizeConversionMappings(mappings)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	if err := q.DeleteConversionMappingsByCampaign(ctx, domainToPgUUID(campaignID)); err != nil {
		return nil, err
	}
	for i := range normalized {
		row := &normalized[i]
		if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
			CampaignID:    domainToPgUUID(campaignID),
			InboundStatus: row.InboundStatus,
			GoalName:      row.GoalName,
			PayoutMicro:   row.PayoutMicro,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizeConversionMappings(mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	return campaign.NormalizeConversionMappings(mappings)
}

func conversionMappingToDTO(row *db.CampaignConversionMapping) ConversionMappingDTO {
	return campaign.ConversionMappingToDTO(row)
}

func domainToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

type AffiliateStatusPresetDTO struct {
	Name     string                          `json:"name"`
	Statuses []AffiliateStatusPresetEntryDTO `json:"statuses"`
}

type AffiliateStatusPresetEntryDTO struct {
	InboundStatus string `json:"inbound_status"`
	GoalName      string `json:"goal_name"`
}
